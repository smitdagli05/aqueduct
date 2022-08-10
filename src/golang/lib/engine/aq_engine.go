package engine

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"time"

	artifact_db "github.com/aqueducthq/aqueduct/lib/collections/artifact"
	"github.com/aqueducthq/aqueduct/lib/collections/artifact_result"
	"github.com/aqueducthq/aqueduct/lib/collections/notification"
	operator_db "github.com/aqueducthq/aqueduct/lib/collections/operator"
	"github.com/aqueducthq/aqueduct/lib/collections/operator_result"
	"github.com/aqueducthq/aqueduct/lib/collections/shared"
	"github.com/aqueducthq/aqueduct/lib/collections/user"
	"github.com/aqueducthq/aqueduct/lib/collections/workflow"
	"github.com/aqueducthq/aqueduct/lib/collections/workflow_dag"
	"github.com/aqueducthq/aqueduct/lib/collections/workflow_dag_edge"
	"github.com/aqueducthq/aqueduct/lib/collections/workflow_dag_result"
	"github.com/aqueducthq/aqueduct/lib/collections/workflow_watcher"
	"github.com/aqueducthq/aqueduct/lib/cronjob"
	"github.com/aqueducthq/aqueduct/lib/database"
	"github.com/aqueducthq/aqueduct/lib/job"
	shared_utils "github.com/aqueducthq/aqueduct/lib/lib_utils"
	"github.com/aqueducthq/aqueduct/lib/vault"
	dag_utils "github.com/aqueducthq/aqueduct/lib/workflow/dag"
	"github.com/aqueducthq/aqueduct/lib/workflow/operator"
	"github.com/aqueducthq/aqueduct/lib/workflow/operator/connector/github"
	workflow_utils "github.com/aqueducthq/aqueduct/lib/workflow/utils"
	"github.com/dropbox/godropbox/errors"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type AqueductTimeConfig struct {
	// Configures exactly long we wait before polling again on an in-progress operator.
	OperatorPollInterval time.Duration

	// Configures the maximum amount of time we wait for execution to finish before aborting the run.
	ExecTimeout time.Duration

	// Configures the maximum amount of time we want for any leftover, in-progress operators to complete,
	// after execution has already finished. Once this time is exceeded, we'll give up.
	CleanupTimeout time.Duration
}

type EngineReaders struct {
	WorkflowReader          workflow.Reader
	WorkflowDagReader       workflow_dag.Reader
	WorkflowDagEdgeReader   workflow_dag_edge.Reader
	WorkflowDagResultReader workflow_dag_result.Reader
	OperatorReader          operator_db.Reader
	OperatorResultReader    operator_result.Reader
	ArtifactReader          artifact_db.Reader
	ArtifactResultReader    artifact_result.Reader
	UserReader              user.Reader
}

type EngineWriters struct {
	WorkflowWriter          workflow.Writer
	WorkflowDagWriter       workflow_dag.Writer
	WorkflowDagEdgeWriter   workflow_dag_edge.Writer
	WorkflowDagResultWriter workflow_dag_result.Writer
	WorkflowWatcherWriter   workflow_watcher.Writer
	OperatorWriter          operator_db.Writer
	OperatorResultWriter    operator_result.Writer
	ArtifactWriter          artifact_db.Writer
	ArtifactResultWriter    artifact_result.Writer
	NotificationWriter      notification.Writer
}

type aqEngine struct {
	Database       database.Database
	GithubManager  github.Manager
	Vault          vault.Vault
	CronjobManager cronjob.CronjobManager
	StorageConfig  *shared.StorageConfig
	AqPath         string

	// Readers and Writers needed for workflow management
	*EngineReaders
	*EngineWriters
}

type workflowRunMetadata struct {
	// Maps every operator to the number of its immediate dependencies
	// that still needs to be computed. When this hits 0 during execution,
	// then the operator is ready to be scheduled.
	OpToDependencyCount map[uuid.UUID]int
	InProgressOps       map[uuid.UUID]operator.Operator
	CompletedOps        map[uuid.UUID]operator.Operator
	Status              shared.ExecutionStatus
}

type WorkflowPreviewResult struct {
	Status    shared.ExecutionStatus
	Operators map[uuid.UUID]shared.ExecutionState
	Artifacts map[uuid.UUID]PreviewArtifactResults
}

type previewFloatArtifactResponse struct {
	Val float64 `json:"val"`
}

type previewBoolArtifactResponse struct {
	Passed bool `json:"passed"`
}

type previewParamArtifactResponse struct {
	Val string `json:"val"`
}

type previewTableArtifactResponse struct {
	TableSchema []map[string]string `json:"table_schema"`
	Data        string              `json:"data"`
}

type PreviewArtifactResults struct {
	Table  *previewTableArtifactResponse `json:"table"`
	Metric *previewFloatArtifactResponse `json:"metric"`
	Check  *previewBoolArtifactResponse  `json:"check"`
	Param  *previewParamArtifactResponse `json:"param"`
}

func NewAqEngine(
	database database.Database,
	githubManager github.Manager,
	vault vault.Vault,
	aqPath string,
	storageConfig *shared.StorageConfig,
	engineReaders *EngineReaders,
	engineWriters *EngineWriters,
) (*aqEngine, error) {
	cronjobManager := cronjob.NewProcessCronjobManager()

	return &aqEngine{
		Database:       database,
		GithubManager:  githubManager,
		Vault:          vault,
		CronjobManager: cronjobManager,
		StorageConfig:  storageConfig,
		AqPath:         aqPath,
		EngineReaders:  engineReaders,
		EngineWriters:  engineWriters,
	}, nil
}

// TODO ENG-1444: Remove jobSpec/ creation once we get rid of executor
func (eng *aqEngine) ScheduleWorkflow(
	ctx context.Context,
	workflowId uuid.UUID,
	name string,
	period string,
) error {
	jobSpec := job.NewWorkflowSpec(
		name,
		workflowId.String(),
		eng.Database.Config(),
		eng.Vault.Config(),
		&job.ProcessConfig{
			BinaryDir:          path.Join(eng.AqPath, job.BinaryDir),
			OperatorStorageDir: path.Join(eng.AqPath, job.OperatorStorageDir),
		},
		eng.GithubManager.Config(),
		eng.AqPath,
		eng.StorageConfig,
		nil,
	)
	err := eng.CronjobManager.DeployCronJob(
		ctx,
		name,
		period,
		eng.generateCronFunction(name, jobSpec),
	)
	if err != nil {
		return errors.Wrap(err, "Unable to schedule workflow.")
	}
	return nil
}

func (eng *aqEngine) ExecuteWorkflow(
	ctx context.Context,
	workflowId uuid.UUID,
	timeConfig *AqueductTimeConfig,
	parameters map[string]string,
) (shared.ExecutionStatus, error) {
	// TODO: Generalize JobManager type from user input.
	// engineJobManager depends on the type of engine used.
	engineJobManager, err := job.NewJobManager(
		&job.ProcessConfig{
			BinaryDir:          path.Join(eng.AqPath, job.BinaryDir),
			OperatorStorageDir: path.Join(eng.AqPath, job.OperatorStorageDir),
		},
	)
	if err != nil {
		return shared.FailedExecutionStatus, errors.Wrap(err, "Unable to create JobManager.")
	}

	dbWorkflowDag, err := workflow_utils.ReadLatestWorkflowDagFromDatabase(
		ctx,
		workflowId,
		eng.WorkflowReader,
		eng.WorkflowDagReader,
		eng.OperatorReader,
		eng.ArtifactReader,
		eng.WorkflowDagEdgeReader,
		eng.Database,
	)
	if err != nil {
		return shared.FailedExecutionStatus, errors.Wrap(err, "Error reading latest workflowDag.")
	}

	githubClient, err := eng.GithubManager.GetClient(ctx, dbWorkflowDag.Metadata.UserId)
	if err != nil {
		return shared.FailedExecutionStatus, errors.Wrap(err, "Error getting github client.")
	}

	dbWorkflowDag, err = workflow_utils.UpdateWorkflowDagToLatest(
		ctx,
		githubClient,
		dbWorkflowDag,
		eng.WorkflowReader,
		eng.WorkflowWriter,
		eng.WorkflowDagReader,
		eng.WorkflowDagWriter,
		eng.OperatorReader,
		eng.OperatorWriter,
		eng.WorkflowDagEdgeReader,
		eng.WorkflowDagEdgeWriter,
		eng.ArtifactReader,
		eng.ArtifactWriter,
		eng.Database,
	)
	if err != nil {
		return shared.FailedExecutionStatus, errors.Wrap(err, "Error updating workflowDag to latest.")
	}

	for name, newVal := range parameters {
		op := dbWorkflowDag.GetOperatorByName(name)
		if op == nil {
			continue
		}
		if !op.Spec.IsParam() {
			return shared.FailedExecutionStatus, errors.Wrap(err, "Cannot set parameters on a non-parameter operator.")
		}
		dbWorkflowDag.Operators[op.Id].Spec.Param().Val = newVal
	}

	dag, err := dag_utils.NewWorkflowDag(
		ctx,
		dbWorkflowDag,
		eng.WorkflowDagResultWriter,
		eng.OperatorResultWriter,
		eng.ArtifactResultWriter,
		eng.WorkflowReader,
		eng.NotificationWriter,
		eng.UserReader,
		engineJobManager,
		eng.Vault,
		eng.StorageConfig,
		false, // is not preview
		eng.Database,
	)
	if err != nil {
		return shared.FailedExecutionStatus, errors.Wrap(err, "Unable to create NewWorkflowDag.")
	}

	opToDependencyCount := make(map[uuid.UUID]int, len(dag.Operators()))
	for _, op := range dag.Operators() {
		inputs, err := dag.OperatorInputs(op)
		if err != nil {
			return shared.FailedExecutionStatus, errors.Wrap(err, "Unable to initialize operator inputs.")
		}
		opToDependencyCount[op.ID()] = len(inputs)
	}

	workflowRunMetadata := &workflowRunMetadata{
		OpToDependencyCount: opToDependencyCount,
		InProgressOps:       make(map[uuid.UUID]operator.Operator, len(dag.Operators())),
		CompletedOps:        make(map[uuid.UUID]operator.Operator, len(dag.Operators())),
		Status:              shared.PendingExecutionStatus,
	}

	// Make sure to persist the dag results on exit.
	defer func() {
		log.Info("workflowRunMetadata: ")
		log.Info(workflowRunMetadata)
		err = dag.PersistResult(ctx, workflowRunMetadata.Status)
		if err != nil {
			log.Errorf("Error when persisting dag results: %v", err)
		}
	}()

	err = dag.InitializeResults(ctx)
	if err != nil {
		return shared.FailedExecutionStatus, errors.Wrap(err, "Unable to initialize dag results.")
	}

	workflowRunMetadata.Status = shared.RunningExecutionStatus
	err = eng.execute(
		ctx,
		dag,
		workflowRunMetadata,
		timeConfig,
		engineJobManager,
		true, // should persist results
	)
	if err != nil {
		workflowRunMetadata.Status = shared.FailedExecutionStatus
		return shared.FailedExecutionStatus, errors.Wrap(err, "Error when executing workflow.")
	} else {
		workflowRunMetadata.Status = shared.SucceededExecutionStatus
	}

	return shared.SucceededExecutionStatus, nil
}

func (eng *aqEngine) PreviewWorkflow(
	ctx context.Context,
	dbWorkflowDag *workflow_dag.DBWorkflowDag,
	timeConfig *AqueductTimeConfig,
) (*WorkflowPreviewResult, error) {
	// previewing workflows always happens using the ProcessJobManager
	jobManager, err := job.NewProcessJobManager(
		&job.ProcessConfig{
			BinaryDir:          path.Join(eng.AqPath, job.BinaryDir),
			OperatorStorageDir: path.Join(eng.AqPath, job.OperatorStorageDir),
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create JobManager.")
	}

	dag, err := dag_utils.NewWorkflowDag(
		ctx,
		dbWorkflowDag,
		eng.WorkflowDagResultWriter,
		eng.OperatorResultWriter,
		eng.ArtifactResultWriter,
		eng.WorkflowReader,
		eng.NotificationWriter,
		eng.UserReader,
		jobManager,
		eng.Vault,
		eng.StorageConfig,
		true, // is a preview
		eng.Database,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create NewWorkflowDag.")
	}

	defer eng.cleanupWorkflow(ctx, dag)

	opToDependencyCount := make(map[uuid.UUID]int, len(dag.Operators()))
	for _, op := range dag.Operators() {
		inputs, err := dag.OperatorInputs(op)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to initialize operator inputs.")
		}
		opToDependencyCount[op.ID()] = len(inputs)
	}

	workflowRunMetadata := &workflowRunMetadata{
		OpToDependencyCount: opToDependencyCount,
		InProgressOps:       make(map[uuid.UUID]operator.Operator, len(dag.Operators())),
		CompletedOps:        make(map[uuid.UUID]operator.Operator, len(dag.Operators())),
		Status:              shared.PendingExecutionStatus,
	}

	workflowRunMetadata.Status = shared.RunningExecutionStatus
	err = eng.execute(
		ctx,
		dag,
		workflowRunMetadata,
		timeConfig,
		jobManager,
		false, // should not persist results
	)
	if err != nil {
		workflowRunMetadata.Status = shared.FailedExecutionStatus
	} else {
		workflowRunMetadata.Status = shared.SucceededExecutionStatus
	}

	execStateByOp := make(map[uuid.UUID]shared.ExecutionState, len(dag.Operators()))
	for _, op := range dag.Operators() {
		execState, err := op.GetExecState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get operator execution state.")
		}
		execStateByOp[op.ID()] = *execState
	}

	// Only include artifact results that were successfully computed.
	artifactResults := make(map[uuid.UUID]PreviewArtifactResults)
	for _, artf := range dag.Artifacts() {
		if artf.Computed(ctx) {
			artifactResp, err := convertToPreviewArtifactResponse(ctx, artf)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to convert artifact result.")
			}
			artifactResults[artf.ID()] = *artifactResp
		}
	}

	return &WorkflowPreviewResult{
		Status:    workflowRunMetadata.Status,
		Operators: execStateByOp,
		Artifacts: artifactResults,
	}, nil
}

func (eng *aqEngine) DeleteWorkflow(
	ctx context.Context,
	workflowId uuid.UUID,
) error {
	txn, err := eng.Database.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "Unable to delete workflow.")
	}
	defer database.TxnRollbackIgnoreErr(ctx, txn)

	// We first retrieve all relevant records from the database.
	workflowObject, err := eng.WorkflowReader.GetWorkflow(ctx, workflowId, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while retrieving workflow.")
	}

	workflowDagsToDelete, err := eng.WorkflowDagReader.GetWorkflowDagsByWorkflowId(ctx, workflowObject.Id, txn)
	if err != nil || len(workflowDagsToDelete) == 0 {
		return errors.Wrap(err, "Unexpected error occurred while retrieving workflow dags.")
	}

	workflowDagIds := make([]uuid.UUID, 0, len(workflowDagsToDelete))
	for _, workflowDag := range workflowDagsToDelete {
		workflowDagIds = append(workflowDagIds, workflowDag.Id)
	}

	workflowDagResultsToDelete, err := eng.WorkflowDagResultReader.GetWorkflowDagResultsByWorkflowId(ctx, workflowObject.Id, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while retrieving workflow dag results.")
	}

	workflowDagResultIds := make([]uuid.UUID, 0, len(workflowDagResultsToDelete))
	for _, workflowDagResult := range workflowDagResultsToDelete {
		workflowDagResultIds = append(workflowDagResultIds, workflowDagResult.Id)
	}

	workflowDagEdgesToDelete, err := eng.WorkflowDagEdgeReader.GetEdgesByWorkflowDagIds(ctx, workflowDagIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while retrieving workflow dag edges.")
	}

	operatorIds := make([]uuid.UUID, 0, len(workflowDagEdgesToDelete))
	artifactIds := make([]uuid.UUID, 0, len(workflowDagEdgesToDelete))

	operatorIdMap := make(map[uuid.UUID]bool)
	artifactIdMap := make(map[uuid.UUID]bool)

	for _, workflowDagEdge := range workflowDagEdgesToDelete {
		var operatorId uuid.UUID
		var artifactId uuid.UUID

		if workflowDagEdge.Type == workflow_dag_edge.OperatorToArtifactType {
			operatorId = workflowDagEdge.FromId
			artifactId = workflowDagEdge.ToId
		} else {
			operatorId = workflowDagEdge.ToId
			artifactId = workflowDagEdge.FromId
		}

		if _, ok := operatorIdMap[operatorId]; !ok {
			operatorIdMap[operatorId] = true
			operatorIds = append(operatorIds, operatorId)
		}

		if _, ok := artifactIdMap[artifactId]; !ok {
			artifactIdMap[artifactId] = true
			artifactIds = append(artifactIds, artifactId)
		}
	}

	operatorsToDelete, err := eng.OperatorReader.GetOperators(ctx, operatorIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while retrieving operators.")
	}

	operatorResultsToDelete, err := eng.OperatorResultReader.GetOperatorResultsByWorkflowDagResultIds(
		ctx,
		workflowDagResultIds,
		txn,
	)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while retrieving operator results.")
	}

	operatorResultIds := make([]uuid.UUID, 0, len(operatorResultsToDelete))
	for _, operatorResult := range operatorResultsToDelete {
		operatorResultIds = append(operatorResultIds, operatorResult.Id)
	}

	artifactResultsToDelete, err := eng.ArtifactResultReader.GetArtifactResultsByWorkflowDagResultIds(
		ctx,
		workflowDagResultIds,
		txn,
	)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while retrieving artifact results.")
	}

	artifactResultIds := make([]uuid.UUID, 0, len(artifactResultsToDelete))
	for _, artifactResult := range artifactResultsToDelete {
		artifactResultIds = append(artifactResultIds, artifactResult.Id)
	}

	// Start deleting database records.
	err = eng.WorkflowWatcherWriter.DeleteWorkflowWatcherByWorkflowId(ctx, workflowObject.Id, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting workflow watchers.")
	}

	err = eng.OperatorResultWriter.DeleteOperatorResults(ctx, operatorResultIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting operator results.")
	}

	err = eng.ArtifactResultWriter.DeleteArtifactResults(ctx, artifactResultIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting artifact results.")
	}

	err = eng.WorkflowDagResultWriter.DeleteWorkflowDagResults(ctx, workflowDagResultIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting workflow dag results.")
	}

	err = eng.WorkflowDagEdgeWriter.DeleteEdgesByWorkflowDagIds(ctx, workflowDagIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting workflow dag edges.")
	}

	err = eng.OperatorWriter.DeleteOperators(ctx, operatorIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting operators.")
	}

	err = eng.ArtifactWriter.DeleteArtifacts(ctx, artifactIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting artifacts.")
	}

	err = eng.WorkflowDagWriter.DeleteWorkflowDags(ctx, workflowDagIds, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting workflow dags.")
	}

	err = eng.WorkflowWriter.DeleteWorkflow(ctx, workflowObject.Id, txn)
	if err != nil {
		return errors.Wrap(err, "Unexpected error occurred while deleting workflow.")
	}

	if err := txn.Commit(ctx); err != nil {
		return errors.Wrap(err, "Failed to delete workflow.")
	}

	// Delete storage files (artifact content and function files)
	storagePaths := make([]string, 0, len(operatorIds)+len(artifactResultIds))
	for _, op := range operatorsToDelete {
		if op.Spec.IsFunction() || op.Spec.IsMetric() || op.Spec.IsCheck() {
			storagePaths = append(storagePaths, op.Spec.Function().StoragePath)
		}
	}

	for _, art := range artifactResultsToDelete {
		storagePaths = append(storagePaths, art.ContentPath)
	}

	// Note: for now we assume all workflow dags have the same storage config.
	// This assumption will stay true until we allow users to configure custom storage config to store stuff.
	storageConfig := workflowDagsToDelete[0].StorageConfig
	for _, workflowDag := range workflowDagsToDelete {
		if !reflect.DeepEqual(workflowDag.StorageConfig, storageConfig) {
			return errors.New("Workflow Dags have mismatching storage config.")
		}
	}

	workflow_utils.CleanupStorageFiles(ctx, &storageConfig, storagePaths)

	// Delete the cron job if it had one.
	if workflowObject.Schedule.CronSchedule != "" {
		cronjobName := shared_utils.AppendPrefix(workflowObject.Id.String())
		err = eng.CronjobManager.DeleteCronJob(ctx, cronjobName)
		if err != nil {
			return errors.Wrap(err, "Failed to delete workflow's cronjob.")
		}
	}
	return nil
}

func (eng *aqEngine) EditWorkflow(
	ctx context.Context,
	txn database.Database,
	workflowId uuid.UUID,
	workflowName string,
	workflowDescription string,
	schedule *workflow.Schedule,
) error {
	changes := map[string]interface{}{}
	if workflowName != "" {
		changes["name"] = workflowName
	}

	if workflowDescription != "" {
		changes["description"] = workflowDescription
	}

	if schedule.Trigger != "" {
		cronjobName := shared_utils.AppendPrefix(workflowId.String())
		err := eng.updateWorkflowSchedule(ctx, workflowId, cronjobName, schedule)
		if err != nil {
			return errors.Wrap(err, "Unable to update workflow schedule.")
		}
		changes["schedule"] = schedule
	}

	_, err := eng.WorkflowWriter.UpdateWorkflow(ctx, workflowId, changes, txn)
	if err != nil {
		return errors.Wrap(err, "Unable to update workflow.")
	}

	return nil
}

// TODO ENG-1444: This function is only used to trigger a Workflow.
// Remove once executor is done.
func (eng *aqEngine) TriggerWorkflow(
	ctx context.Context,
	workflowId uuid.UUID,
	name string,
	timeConfig *AqueductTimeConfig,
	parameters map[string]string,
) (shared.ExecutionStatus, error) {
	jobManager, err := job.NewProcessJobManager(
		&job.ProcessConfig{
			BinaryDir:          path.Join(eng.AqPath, job.BinaryDir),
			OperatorStorageDir: path.Join(eng.AqPath, job.OperatorStorageDir),
		},
	)
	if err != nil {
		log.Errorf("Unable to create JobManager: %v", err)
	}

	jobSpec := job.NewWorkflowSpec(
		name,
		workflowId.String(),
		eng.Database.Config(),
		eng.Vault.Config(),
		&job.ProcessConfig{
			BinaryDir:          path.Join(eng.AqPath, job.BinaryDir),
			OperatorStorageDir: path.Join(eng.AqPath, job.OperatorStorageDir),
		},
		eng.GithubManager.Config(),
		eng.AqPath,
		eng.StorageConfig,
		parameters,
	)

	jobName := fmt.Sprintf("%s-%d", name, time.Now().Unix())
	err = jobManager.Launch(context.Background(), jobName, jobSpec)
	if err != nil {
		log.Errorf("Error running job %s: %v", jobName, err)
		return shared.FailedExecutionStatus, errors.Wrap(err, "Error running job.")
	} else {
		log.Infof("Launched job %s", jobName)
		return shared.PendingExecutionStatus, nil
	}
}

func (eng *aqEngine) cleanupWorkflow(ctx context.Context, workflowDag dag_utils.WorkflowDag) {
	for _, op := range workflowDag.Operators() {
		op.Finish(ctx)
	}
}

func (eng *aqEngine) execute(
	ctx context.Context,
	workflowDag dag_utils.WorkflowDag,
	workflowRunMetadata *workflowRunMetadata,
	timeConfig *AqueductTimeConfig,
	jobManager job.JobManager,
	shouldPersistResults bool,
) error {
	// These are the operators of immediate interest. They either need to be scheduled or polled on.
	inProgressOps := workflowRunMetadata.InProgressOps
	completedOps := workflowRunMetadata.CompletedOps
	dag := workflowDag
	opToDependencyCount := workflowRunMetadata.OpToDependencyCount

	// Kick off execution by starting all operators that don't have any inputs.
	for _, op := range dag.Operators() {
		if opToDependencyCount[op.ID()] == 0 {
			inProgressOps[op.ID()] = op
		}
	}

	if len(inProgressOps) == 0 {
		return errors.Newf("No initial operators to schedule.")
	}

	// Wait a little bit for all active operators to finish before exiting on failure.
	defer waitForInProgressOperators(ctx, inProgressOps, timeConfig.OperatorPollInterval, timeConfig.CleanupTimeout)

	start := time.Now()

	for len(inProgressOps) > 0 {
		if time.Since(start) > timeConfig.ExecTimeout {
			return errors.New("Reached timeout waiting for workflow to complete.")
		}

		for _, op := range inProgressOps {
			execState, err := op.GetExecState(ctx)
			if err != nil {
				return err
			}

			if execState.Status == shared.PendingExecutionStatus {
				spec := op.JobSpec()
				err = jobManager.Launch(ctx, spec.JobName(), spec)
				if err != nil {
					return errors.Wrapf(err, "Unable to schedule operator %s.", op.Name())
				}
				continue
			} else if execState.Status == shared.RunningExecutionStatus {
				continue
			}
			if execState.Status != shared.FailedExecutionStatus && execState.Status != shared.SucceededExecutionStatus {
				return errors.Newf("Internal error: the operator is expected to have terminated, but instead has status %s", execState.Status)
			}

			// From here on we can assume that the operator has terminated.
			if shouldPersistResults {
				err = op.PersistResult(ctx)
				if err != nil {
					return errors.Wrapf(err, "Error when finishing execution of operator %s", op.Name())
				}
			}

			// We can continue orchestration on non-fatal errors.
			if shouldStopExecution(execState) {
				return opFailureError(*execState.FailureType, op)
			}

			// Add the operator to the completed stack, and remove it from the in-progress one.
			if _, ok := completedOps[op.ID()]; ok {
				return errors.Newf("Internal error: operator %s was completed twice.", op.Name())
			}
			completedOps[op.ID()] = op
			delete(inProgressOps, op.ID())

			outputArtifacts, err := dag.OperatorOutputs(op)
			if err != nil {
				return err
			}
			for _, outputArtifact := range outputArtifacts {
				nextOps, err := dag.OperatorsOnArtifact(outputArtifact)
				if err != nil {
					return err
				}

				for _, nextOp := range nextOps {
					// Decrement the active dependency count for every downstream operator.
					// Once this count reaches zero, we can schedule the next operator.
					opToDependencyCount[nextOp.ID()] -= 1

					if opToDependencyCount[nextOp.ID()] < 0 {
						return errors.Newf("Internal error: operator %s has a negative dependnecy count.", op.Name())
					}

					if opToDependencyCount[nextOp.ID()] == 0 {
						// Defensive check: do not reschedule an already in-progress operator. This shouldn't actually
						// matter because we only keep and update a single copy an on operator.
						if _, ok := inProgressOps[nextOp.ID()]; !ok {
							inProgressOps[nextOp.ID()] = nextOp
						}
					}
				}
			}

			time.Sleep(timeConfig.OperatorPollInterval)
		}
	}

	if len(completedOps) != len(dag.Operators()) {
		return errors.Newf("Internal error: %d operators were provided but only %d completed.", len(dag.Operators()), len(completedOps))
	}

	for opID, depCount := range opToDependencyCount {
		if depCount != 0 {
			return errors.Newf("Internal error: operator %s has a non-zero dep count %d.", opID, depCount)
		}
	}
	return nil
}

func (eng *aqEngine) generateCronFunction(name string, jobSpec job.Spec) func() {
	// TODO ENG-1444: Creating this process job manager just to
	// launch the executor which calls ExecuteWorkflow().
	// Replace with call to ExecuteWorkflow() once executor is removed.
	jobManager, err := job.NewProcessJobManager(
		&job.ProcessConfig{
			BinaryDir:          path.Join(eng.AqPath, job.BinaryDir),
			OperatorStorageDir: path.Join(eng.AqPath, job.OperatorStorageDir),
		},
	)
	if err != nil {
		log.Errorf("Unable to create JobManager: %v", err)
	}

	return func() {
		jobName := fmt.Sprintf("%s-%d", name, time.Now().Unix())
		err := jobManager.Launch(context.Background(), jobName, jobSpec)
		if err != nil {
			log.Errorf("Error running cron job %s: %v", jobName, err)
		} else {
			log.Infof("Launched cron job %s", jobName)
		}
	}
}

func (eng *aqEngine) updateWorkflowSchedule(
	ctx context.Context,
	workflowId uuid.UUID,
	cronjobName string,
	newSchedule *workflow.Schedule,
) error {
	// How we update the workflow schedule depends on whether a cron job already exists.
	// A manually triggered workflow does not have a cron job. If we're editing it to have a periodic
	// schedule, we'll need to create a new cron job.
	if !eng.CronjobManager.CronJobExists(ctx, cronjobName) {
		if newSchedule.CronSchedule != "" {

			err := eng.ScheduleWorkflow(
				ctx,
				workflowId,
				cronjobName,
				string(newSchedule.CronSchedule),
			)
			if err != nil {
				return errors.Wrap(err, "Unable to deploy new cron job.")
			}
		}
		// We will no-op if the workflow continues to be manually triggered.
	} else {
		// Here, we can blindly set the cron job to be paused without any other
		// modification. The pausedness of the workflow will be written to the
		// database by the changes map above, and `prepare` guarantees us that
		// if `Paused` is true, then the workflow type is `Periodic`, which in
		// turn means a schedule must be set.
		newCronSchedule := string(newSchedule.CronSchedule)
		if newSchedule.Paused {
			// The `EditCronJob` helper automatically pauses a workflow when
			// you set the cron job schedule to an empty string.
			newCronSchedule = ""
		}
		// TODO ENG-1444: Remove jobSpec once executor is removed.
		jobSpec := job.NewWorkflowSpec(
			cronjobName,
			workflowId.String(),
			eng.Database.Config(),
			eng.Vault.Config(),
			&job.ProcessConfig{
				BinaryDir:          path.Join(eng.AqPath, job.BinaryDir),
				OperatorStorageDir: path.Join(eng.AqPath, job.OperatorStorageDir),
			},
			eng.GithubManager.Config(),
			eng.AqPath,
			eng.StorageConfig,
			nil,
		)

		err := eng.CronjobManager.EditCronJob(
			ctx,
			cronjobName,
			newCronSchedule,
			eng.generateCronFunction(cronjobName, jobSpec),
		)
		if err != nil {
			return errors.Wrap(err, "Unable to change workflow schedule.")
		}
	}
	return nil
}