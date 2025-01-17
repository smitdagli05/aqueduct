package job

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/aqueducthq/aqueduct/lib/models/shared"
	"github.com/dropbox/godropbox/errors"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"
)

const (
	defaultPythonExecutorPackage = "aqueduct_executor"
	connectorPythonPath          = "operators.connectors.data.main"
	paramPythonPath              = "operators.param_executor.main"
	systemMetricPythonPath       = "operators.system_metric_executor.main"
	compileAirflowPythonPath     = "operators.airflow.main"
	executorBinary               = "executor"
	functionExecutorBashScript   = "start-function-executor.sh"

	processRunningStatus = "R"

	BinaryDir          = "bin/"
	OperatorStorageDir = "storage/operators/"
	LogsDir            = "logs/"
)

var (
	defaultBinaryDir          = path.Join(os.Getenv("HOME"), ".aqueduct", BinaryDir)
	defaultOperatorStorageDir = path.Join(os.Getenv("HOME"), ".aqueduct", OperatorStorageDir)
	defaultLogsDir            = path.Join(os.Getenv("HOME"), ".aqueduct", "server", LogsDir)
)

type Command struct {
	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

type cronMetadata struct {
	// If the cronJob is nil, it means the corresponding workflow has been paused.
	cronJob *gocron.Job
	// We need to store the job spec because when the workflow is resumed from the pause state, we
	// need the spec to redeploy the cron job.
	jobSpec Spec
}

// Please use thread-safe read / insert / remove APIs to maintain maps.
// These APIs are wrapped with proper locks to support concurrency.
// Never try to access map using go's native APIs.
type ProcessJobManager struct {
	conf          *ProcessConfig
	cmds          map[string]*Command
	cronScheduler *gocron.Scheduler
	// A mapping from cron job name to cron job object pointer.
	cronMapping  map[string]*cronMetadata
	condaEnvName string
	cmdMutex     *sync.RWMutex
	cronMutex    *sync.RWMutex
}

func (j *ProcessJobManager) getCmd(key string) (*Command, bool) {
	j.cmdMutex.RLock()
	cmd, ok := j.cmds[key]
	j.cmdMutex.RUnlock()
	return cmd, ok
}

func (j *ProcessJobManager) setCmd(key string, cmd *Command) {
	j.cmdMutex.Lock()
	j.cmds[key] = cmd
	j.cmdMutex.Unlock()
}

func (j *ProcessJobManager) deleteCmd(key string) {
	j.cmdMutex.Lock()
	delete(j.cmds, key)
	j.cmdMutex.Unlock()
}

func (j *ProcessJobManager) getCronMap(key string) (*cronMetadata, bool) {
	j.cronMutex.RLock()
	cron, ok := j.cronMapping[key]
	j.cronMutex.RUnlock()
	return cron, ok
}

func (j *ProcessJobManager) setCronMap(key string, cron *cronMetadata) {
	j.cronMutex.Lock()
	j.cronMapping[key] = cron
	j.cronMutex.Unlock()
}

func (j *ProcessJobManager) deleteCronMap(key string) {
	j.cronMutex.Lock()
	delete(j.cronMapping, key)
	j.cronMutex.Unlock()
}

func NewProcessJobManager(conf *ProcessConfig) (*ProcessJobManager, error) {
	if conf.PythonExecutorPackage == "" {
		conf.PythonExecutorPackage = defaultPythonExecutorPackage
	}

	if conf.BinaryDir == "" {
		conf.BinaryDir = defaultBinaryDir
	}

	if conf.OperatorStorageDir == "" {
		conf.OperatorStorageDir = defaultOperatorStorageDir
	}

	cronScheduler := gocron.NewScheduler(time.UTC)
	cronScheduler.StartAsync()

	return &ProcessJobManager{
		conf:          conf,
		cmds:          map[string]*Command{},
		cronScheduler: cronScheduler,
		cronMapping:   map[string]*cronMetadata{},
		cmdMutex:      &sync.RWMutex{},
		cronMutex:     &sync.RWMutex{},
		condaEnvName:  conf.CondaEnvName,
	}, nil
}

func (j *ProcessJobManager) mapJobTypeToCmd(jobName string, spec Spec) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	if spec.Type() == WorkflowJobType {
		workflowSpec, ok := spec.(*WorkflowSpec)
		if !ok {
			return nil, errors.New("Unable to cast job spec to WorkflowSpec.")
		}

		specStr, err := EncodeSpec(workflowSpec, GobSerializationType)
		if err != nil {
			return nil, err
		}

		logFilePath := path.Join(defaultLogsDir, jobName)
		log.Infof("Logs for job %s are stored in %s", jobName, logFilePath)

		cmd = exec.Command(
			fmt.Sprintf("%s/%s", j.conf.BinaryDir, executorBinary),
			"--spec",
			specStr,
			"--logs-path",
			logFilePath,
		)
	} else if spec.Type() == WorkflowRetentionType {
		workflowRetentionSpec, ok := spec.(*WorkflowRetentionSpec)
		if !ok {
			return nil, errors.New("Unable to cast job spec to workflowRetentionSpec.")
		}

		specStr, err := EncodeSpec(workflowRetentionSpec, GobSerializationType)
		if err != nil {
			return nil, err
		}

		logFilePath := path.Join(defaultLogsDir, jobName)
		log.Infof("Logs for job %s are stored in %s", jobName, logFilePath)

		cmd = exec.Command(
			fmt.Sprintf("%s/%s", j.conf.BinaryDir, executorBinary),
			"--spec",
			specStr,
			"--logs-path",
			logFilePath,
		)
	} else if spec.Type() == DynamicTeardownType {
		dynamicTeardownSpec, ok := spec.(*DynamicTeardownSpec)
		if !ok {
			return nil, errors.New("Unable to cast job spec to dynamicTeardownSpec.")
		}

		specStr, err := EncodeSpec(dynamicTeardownSpec, GobSerializationType)
		if err != nil {
			return nil, err
		}

		logFilePath := path.Join(defaultLogsDir, jobName)
		log.Infof("Logs for job %s are stored in %s", jobName, logFilePath)

		cmd = exec.Command(
			fmt.Sprintf("%s/%s", j.conf.BinaryDir, executorBinary),
			"--spec",
			specStr,
			"--logs-path",
			logFilePath,
		)
	} else if spec.Type() == FunctionJobType {
		functionSpec, ok := spec.(*FunctionSpec)
		if !ok {
			return nil, ErrInvalidJobSpec
		}

		functionSpec.FunctionExtractPath = path.Join(j.conf.OperatorStorageDir, uuid.New().String())
		specStr, err := EncodeSpec(spec, JsonSerializationType)
		if err != nil {
			return nil, err
		}

		if j.condaEnvName != "" {
			cmd = exec.Command(
				"conda",
				"run",
				"-n",
				j.condaEnvName,
				"bash",
				filepath.Join(j.conf.BinaryDir, functionExecutorBashScript),
				specStr,
			)
		} else {
			cmd = exec.Command(
				"bash",
				filepath.Join(j.conf.BinaryDir, functionExecutorBashScript),
				specStr,
			)
		}
	} else if spec.Type() == ParamJobType {
		specStr, err := EncodeSpec(spec, JsonSerializationType)
		if err != nil {
			return nil, err
		}

		cmd = exec.Command(
			"python3",
			"-m",
			fmt.Sprintf("%s.%s", j.conf.PythonExecutorPackage, paramPythonPath),
			"--spec",
			specStr,
		)
	} else if spec.Type() == AuthenticateJobType ||
		spec.Type() == LoadJobType ||
		spec.Type() == ExtractJobType ||
		spec.Type() == LoadTableJobType ||
		spec.Type() == DeleteSavedObjectsJobType ||
		spec.Type() == DiscoverJobType {
		specStr, err := EncodeSpec(spec, JsonSerializationType)
		if err != nil {
			return nil, err
		}

		cmd = exec.Command(
			"python3",
			"-m",
			fmt.Sprintf("%s.%s", j.conf.PythonExecutorPackage, connectorPythonPath),
			"--spec",
			specStr,
		)

		// This is required for credential related temp file creation.
		// See S3 python executor for more details.
		cmd.Dir = j.conf.OperatorStorageDir
	} else if spec.Type() == SystemMetricJobType {
		specStr, err := EncodeSpec(spec, JsonSerializationType)
		if err != nil {
			return nil, err
		}

		return exec.Command(
			"python3",
			"-m",
			fmt.Sprintf("%s.%s", j.conf.PythonExecutorPackage, systemMetricPythonPath),
			"--spec",
			specStr,
		), nil
	} else if spec.Type() == CompileAirflowJobType {
		specStr, err := EncodeSpec(spec, JsonSerializationType)
		if err != nil {
			return nil, err
		}

		return exec.Command(
			"python3",
			"-m",
			fmt.Sprintf("%s.%s", j.conf.PythonExecutorPackage, compileAirflowPythonPath),
			"--spec",
			specStr,
		), nil
	} else {
		return nil, errors.New("Unsupported JobType was passed in.")
	}
	log.Infof("Running job with command: %s", cmd.String())
	return cmd, nil
}

func (j *ProcessJobManager) generateCronFunction(name string, jobSpec Spec) func() {
	return func() {
		jobName := fmt.Sprintf("%s-%d", name, time.Now().Unix())
		err := j.Launch(context.Background(), jobName, jobSpec)
		if err != nil {
			log.Errorf("Error running cron job %s: %v", jobName, err)
		} else {
			log.Infof("Launched cron job %s", jobName)
		}
	}
}

func (j *ProcessJobManager) Config() Config {
	return j.conf
}

func (j *ProcessJobManager) Launch(
	_ context.Context,
	name string,
	spec Spec,
) JobError {
	log.Infof("Running %s job %s.", spec.Type(), name)
	if _, ok := j.getCmd(name); ok {
		return systemError(errors.Newf("A job with the same name %s already exists.", name))
	}

	cmd, err := j.mapJobTypeToCmd(name, spec)
	if err != nil {
		return systemError(err)
	}
	cmd.Env = os.Environ()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	j.setCmd(name, &Command{
		cmd:    cmd,
		stdout: stdout,
		stderr: stderr,
	})
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Start()
	if err != nil {
		return systemError(err)
	}
	return nil
}

func (j *ProcessJobManager) Poll(ctx context.Context, name string) (shared.ExecutionStatus, JobError) {
	command, ok := j.getCmd(name)
	if !ok {
		return shared.UnknownExecutionStatus, jobMissingError(errors.Newf("Job %s does not exist.", name))
	}

	proc, err := process.NewProcess(int32(command.cmd.Process.Pid))
	if err != nil {
		return shared.UnknownExecutionStatus, systemError(err)
	}

	status, err := proc.Status()
	if err != nil {
		return shared.UnknownExecutionStatus, systemError(err)
	}

	if status == processRunningStatus {
		return shared.RunningExecutionStatus, nil
	}

	err = command.cmd.Wait()
	// After wait, we are done with this job and already consumed all of its output, so we garbage
	// collect the entry in j.cmds.
	defer j.deleteCmd(name)
	if err != nil {
		log.Errorf("Unexpected error occurred while executing job %s: %v. Stdout: \n %s \n Stderr: \n %s",
			name,
			err,
			command.stdout.String(),
			command.stderr.String(),
		)
		return shared.FailedExecutionStatus, nil // nolint:nilerr // Poll() should not error on command execution issues.
	}

	log.Infof("Job %s Stdout:\n %s \n Stderr: \n %s",
		name,
		command.stdout.String(),
		command.stderr.String(),
	)
	return shared.SucceededExecutionStatus, nil
}

func (j *ProcessJobManager) DeployCronJob(
	ctx context.Context,
	name string,
	period string,
	spec Spec,
) JobError {
	if _, ok := j.getCronMap(name); ok {
		return systemError(errors.Newf("Cron job with name %s already exists", name))
	}

	cron := &cronMetadata{
		cronJob: nil,
		jobSpec: spec,
	}

	j.setCronMap(name, cron)

	if period != "" {
		cronJob, err := j.cronScheduler.Cron(period).Do(j.generateCronFunction(name, spec))
		if err != nil {
			return systemError(err)
		}

		cron.cronJob = cronJob
	}

	return nil
}

func (j *ProcessJobManager) CronJobExists(ctx context.Context, name string) bool {
	_, ok := j.getCronMap(name)
	return ok
}

func (j *ProcessJobManager) EditCronJob(ctx context.Context, name string, cronString string) JobError {
	cronMetadata, ok := j.getCronMap(name)
	if !ok {
		return systemError(errors.New("Cron job not found"))
	} else {
		if cronMetadata.cronJob == nil {
			// This means the current cron job is paused.
			if cronString == "" {
				return systemError(errors.Newf("Attempting to pause an already paused cron job %s", name))
			}

			cronJob, err := j.cronScheduler.Cron(cronString).Do(j.generateCronFunction(name, cronMetadata.jobSpec))
			if err != nil {
				return systemError(err)
			}

			cronMetadata.cronJob = cronJob
		} else {
			if cronString == "" {
				// This means we want to pause the cron job.
				j.cronScheduler.RemoveByReference(cronMetadata.cronJob)
				cronMetadata.cronJob = nil
			} else {
				_, err := j.cronScheduler.Job(cronMetadata.cronJob).Cron(cronString).Update()
				if err != nil {
					return systemError(err)
				}
			}
		}
		return nil
	}
}

func (j *ProcessJobManager) DeleteCronJob(ctx context.Context, name string) JobError {
	cronMetadata, ok := j.getCronMap(name)
	if ok {
		j.cronScheduler.RemoveByReference(cronMetadata.cronJob)
		j.deleteCronMap(name)
	}

	return nil
}
