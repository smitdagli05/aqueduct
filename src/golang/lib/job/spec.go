package job

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"

	"github.com/aqueducthq/aqueduct/lib/database"
	"github.com/aqueducthq/aqueduct/lib/models/shared"
	"github.com/aqueducthq/aqueduct/lib/models/shared/operator"
	"github.com/aqueducthq/aqueduct/lib/models/shared/operator/check"
	"github.com/aqueducthq/aqueduct/lib/models/shared/operator/connector"
	"github.com/aqueducthq/aqueduct/lib/models/shared/operator/param"
	"github.com/aqueducthq/aqueduct/lib/workflow/operator/connector/auth"
	"github.com/aqueducthq/aqueduct/lib/workflow/operator/connector/github"
	"github.com/dropbox/godropbox/errors"
	"github.com/google/uuid"
)

var ErrInvalidJobSpec = errors.New("Invalid job spec.")

type JobType string

const (
	WorkflowRetentionName = "workflowretentionjob"
	DynamicTeardownName   = "dynamicteardownjob"
)

type SerializationType string

const (
	JsonSerializationType SerializationType = "json"
	GobSerializationType  SerializationType = "gob"
)

const (
	WorkflowJobType           JobType = "workflow"
	FunctionJobType           JobType = "function"
	ParamJobType              JobType = "param"
	SystemMetricJobType       JobType = "system_metric"
	AuthenticateJobType       JobType = "authenticate"
	ExtractJobType            JobType = "extract"
	LoadJobType               JobType = "load"
	LoadTableJobType          JobType = "load-table"
	DeleteSavedObjectsJobType JobType = "delete-saved-objects"
	DiscoverJobType           JobType = "discover"
	WorkflowRetentionType     JobType = "workflow_retention"
	CompileAirflowJobType     JobType = "compile_airflow"
	DynamicTeardownType       JobType = "dynamic_teardown"
)

// `ExecutorConfiguration` represents the configuration variables that are
// used by the Executor cron job responsible for executing transformations
// on datasets.
type ExecutorConfiguration struct {
	Database   *database.DatabaseConfig `yaml:"metadata" json:"metadata"`
	JobManager Config                   `yaml:"jobManager" json:"job_manager"`
}

type Spec interface {
	Type() JobType
	JobName() string
	// HasStorageConfig returns whether this job spec has a storage config.
	HasStorageConfig() bool
	// GetStorageConfig returns the storage config if the job spec has one, otherwise it returns an error.
	GetStorageConfig() (*shared.StorageConfig, error)
}

// BaseSpec defines fields shared by all job specs.
type BaseSpec struct {
	Type JobType `json:"type"  yaml:"type"`
	Name string  `json:"name"  yaml:"name"`
}

func (bs *BaseSpec) JobName() string {
	return bs.Name
}

type DynamicTeardownSpec struct {
	BaseSpec
	ExecutorConfig *ExecutorConfiguration
}

func (dits *DynamicTeardownSpec) HasStorageConfig() bool {
	return false
}

func (dits *DynamicTeardownSpec) GetStorageConfig() (*shared.StorageConfig, error) {
	return nil, errors.New("DynamicTeardown job specs don't have a storage config.")
}

type WorkflowRetentionSpec struct {
	BaseSpec
	ExecutorConfig *ExecutorConfiguration
}

func (wrs *WorkflowRetentionSpec) HasStorageConfig() bool {
	return false
}

func (wrs *WorkflowRetentionSpec) GetStorageConfig() (*shared.StorageConfig, error) {
	return nil, errors.New("WorkflowRetention job specs don't have a storage config.")
}

type WorkflowSpec struct {
	BaseSpec
	WorkflowId     string                 `json:"workflow_id" yaml:"workflowId"`
	GithubManager  github.ManagerConfig   `json:"github_manager" yaml:"github_manager"`
	Parameters     map[string]param.Param `json:"parameters" yaml:"parameters"`
	AqPath         string                 `json:"aq_path" yaml:"aqPath"`
	DisplayIP      string                 `json:"display_ip" yaml:"displayIP"`
	ExecutorConfig *ExecutorConfiguration
}

func (ws *WorkflowSpec) HasStorageConfig() bool {
	return false
}

func (ws *WorkflowSpec) GetStorageConfig() (*shared.StorageConfig, error) {
	return nil, errors.New("Workflow job specs don't have a storage config.")
}

// BasePythonSpec defines fields shared by all Python job specs.
// These Python jobs can be one-off jobs (e.g. Authenticate, Discover)
// or Workflow operators (e.g. Function, Extract, Load).
type BasePythonSpec struct {
	BaseSpec
	StorageConfig shared.StorageConfig `json:"storage_config"  yaml:"storage_config"`
	MetadataPath  string               `json:"metadata_path"  yaml:"metadata_path"`
}

func (bs *BasePythonSpec) HasStorageConfig() bool {
	return true
}

func (bs *BasePythonSpec) GetStorageConfig() (*shared.StorageConfig, error) {
	return &bs.StorageConfig, nil
}

type FunctionSpec struct {
	BasePythonSpec
	FunctionPath                string                   `json:"function_path"  yaml:"function_path"`
	FunctionExtractPath         string                   `json:"function_extract_path" yaml:"function_extract_path"`
	EntryPointFile              string                   `json:"entry_point_file"  yaml:"entry_point_file"`
	EntryPointClass             string                   `json:"entry_point_class"  yaml:"entry_point_class"`
	EntryPointMethod            string                   `json:"entry_point_method"  yaml:"entry_point_method"`
	CustomArgs                  string                   `json:"custom_args"  yaml:"custom_args"`
	InputContentPaths           []string                 `json:"input_content_paths"  yaml:"input_content_paths"`
	InputMetadataPaths          []string                 `json:"input_metadata_paths"  yaml:"input_metadata_paths"`
	OutputContentPaths          []string                 `json:"output_content_paths"  yaml:"output_content_paths"`
	OutputMetadataPaths         []string                 `json:"output_metadata_paths"  yaml:"output_metadata_paths"`
	ExpectedOutputArtifactTypes []string                 `json:"expected_output_artifact_types" yaml:"expected_output_artifact_types"`
	OperatorType                operator.Type            `json:"operator_type" yaml:"operator_type"`
	Resources                   *operator.ResourceConfig `json:"resources" yaml:"resources"`

	// Specific to the check operator. This is left unset by any other function type.
	CheckSeverity *check.Level `json:"check_severity" yaml:"check_severity"`
}

type ParamSpec struct {
	BasePythonSpec
	ExpectedType       shared.ArtifactType `json:"expected_type" yaml:"expected_type"`
	SerializationType  string              `json:"serialization_type" yaml:"serialization_type"`
	OutputContentPath  string              `json:"output_content_path"  yaml:"output_content_path"`
	OutputMetadataPath string              `json:"output_metadata_path"  yaml:"output_metadata_path"`
}

type SystemMetricSpec struct {
	BasePythonSpec
	MetricName         string   `json:"metric_name"  yaml:"metric_name"`
	InputMetadataPaths []string `json:"input_metadata_paths"  yaml:"input_metadata_paths"`
	OutputContentPath  string   `json:"output_content_path"  yaml:"output_content_path"`
	OutputMetadataPath string   `json:"output_metadata_path"  yaml:"output_metadata_path"`
}

type ExtractSpec struct {
	BasePythonSpec
	ConnectorName   shared.Service          `json:"connector_name"  yaml:"connector_name"`
	ConnectorConfig auth.Config             `json:"connector_config"  yaml:"connector_config"`
	Parameters      connector.ExtractParams `json:"parameters"  yaml:"parameters"`

	InputContentPaths  []string `json:"input_content_paths" yaml:"input_content_paths"`
	InputMetadataPaths []string `json:"input_metadata_paths" yaml:"input_metadata_paths"`
	OutputContentPath  string   `json:"output_content_path"  yaml:"output_content_path"`
	OutputMetadataPath string   `json:"output_metadata_path"  yaml:"output_metadata_path"`
}

type DeleteSavedObjectsSpec struct {
	BasePythonSpec
	ConnectorName       map[string]shared.Service `json:"connector_name"  yaml:"connector_name"`
	ConnectorConfig     map[string]auth.Config    `json:"connector_config"  yaml:"connector_config"`
	IntegrationToObject map[string][]string       `json:"integration_to_object"  yaml:"integration_to_object"`
	OutputContentPath   string                    `json:"output_content_path"  yaml:"output_content_path"`
}

type LoadSpec struct {
	BasePythonSpec
	ConnectorName     shared.Service       `json:"connector_name"  yaml:"connector_name"`
	ConnectorConfig   auth.Config          `json:"connector_config"  yaml:"connector_config"`
	Parameters        connector.LoadParams `json:"parameters"  yaml:"parameters"`
	InputContentPath  string               `json:"input_content_path"  yaml:"input_content_path"`
	InputMetadataPath string               `json:"input_metadata_path"  yaml:"input_metadata_path"`
}

type LoadTableSpec struct {
	BasePythonSpec
	ConnectorName   shared.Service `json:"connector_name"  yaml:"connector_name"`
	ConnectorConfig auth.Config    `json:"connector_config"  yaml:"connector_config"`
	CSV             string         `json:"csv"  yaml:"csv"`
	LoadParameters  LoadSpec       `json:"load_parameters"  yaml:"load_parameters"`
}

type AuthenticateSpec struct {
	BasePythonSpec
	ConnectorName   shared.Service `json:"connector_name"  yaml:"connector_name"`
	ConnectorConfig auth.Config    `json:"connector_config"  yaml:"connector_config"`
}

type DiscoverSpec struct {
	BasePythonSpec
	ConnectorName     shared.Service `json:"connector_name"  yaml:"connector_name"`
	ConnectorConfig   auth.Config    `json:"connector_config"  yaml:"connector_config"`
	OutputContentPath string         `json:"output_content_path"  yaml:"output_content_path"`
}

type CompileAirflowSpec struct {
	BasePythonSpec
	WorkflowDagId     uuid.UUID           `json:"workflow_dag_id"  yaml:"workflow_dag_id"`
	OutputContentPath string              `json:"output_content_path"  yaml:"output_content_path"`
	DagId             string              `json:"dag_id"  yaml:"dag_id"`
	CronSchedule      string              `json:"cron_schedule"  yaml:"cron_schedule"`
	TaskSpecs         map[string]Spec     `json:"task_specs"  yaml:"task_specs"`
	TaskEdges         map[string][]string `json:"task_edges"  yaml:"task_edges"`
}

func (*DynamicTeardownSpec) Type() JobType {
	return DynamicTeardownType
}

func (*WorkflowRetentionSpec) Type() JobType {
	return WorkflowRetentionType
}

func (*WorkflowSpec) Type() JobType {
	return WorkflowJobType
}

func (*FunctionSpec) Type() JobType {
	return FunctionJobType
}

func (*ParamSpec) Type() JobType {
	return ParamJobType
}

func (*AuthenticateSpec) Type() JobType {
	return AuthenticateJobType
}

func (*SystemMetricSpec) Type() JobType {
	return SystemMetricJobType
}

func (*ExtractSpec) Type() JobType {
	return ExtractJobType
}

func (*LoadSpec) Type() JobType {
	return LoadJobType
}

func (*LoadTableSpec) Type() JobType {
	return LoadTableJobType
}

func (*DeleteSavedObjectsSpec) Type() JobType {
	return DeleteSavedObjectsJobType
}

func (*DiscoverSpec) Type() JobType {
	return DiscoverJobType
}

func (*CompileAirflowSpec) Type() JobType {
	return CompileAirflowJobType
}

// NewDynamicTeardownJobSpec constructs a Spec for a DynamicTeardownJob.
func NewDynamicTeardownJobSpec(
	database *database.DatabaseConfig,
	jobManager Config,
) Spec {
	return &DynamicTeardownSpec{
		BaseSpec: BaseSpec{
			Type: DynamicTeardownType,
			Name: DynamicTeardownName,
		},

		ExecutorConfig: &ExecutorConfiguration{
			Database:   database,
			JobManager: jobManager,
		},
	}
}

func IsDataType(jobType JobType) bool {
	if jobType == AuthenticateJobType ||
		jobType == LoadJobType ||
		jobType == ExtractJobType ||
		jobType == LoadTableJobType ||
		jobType == DeleteSavedObjectsJobType ||
		jobType == DiscoverJobType {
		return true
	} else {
		return false
	}
}

// NewWorkflowRetentionSpec constructs a Spec for a WorkflowRetentionJob.
func NewWorkflowRetentionJobSpec(
	database *database.DatabaseConfig,
	jobManager Config,
) Spec {
	return &WorkflowRetentionSpec{
		BaseSpec: BaseSpec{
			Type: WorkflowRetentionType,
			Name: WorkflowRetentionName,
		},

		ExecutorConfig: &ExecutorConfiguration{
			Database:   database,
			JobManager: jobManager,
		},
	}
}

// NewWorkflowSpec constructs a Spec for a WorkflowJob.
func NewWorkflowSpec(
	name string,
	workflowId string,
	database *database.DatabaseConfig,
	jobManager Config,
	githubManager github.ManagerConfig,
	aqPath string,
	displayIP string,
	parameters map[string]param.Param,
) Spec {
	return &WorkflowSpec{
		BaseSpec: BaseSpec{
			Type: WorkflowJobType,
			Name: name,
		},
		WorkflowId:    workflowId,
		GithubManager: githubManager,
		AqPath:        aqPath,
		DisplayIP:     displayIP,
		Parameters:    parameters,
		ExecutorConfig: &ExecutorConfiguration{
			Database:   database,
			JobManager: jobManager,
		},
	}
}

func NewBasePythonSpec(
	jobType JobType,
	name string,
	storageConfig shared.StorageConfig,
	metadataPath string,
) BasePythonSpec {
	return BasePythonSpec{
		BaseSpec: BaseSpec{
			Type: jobType,
			Name: name,
		},
		StorageConfig: storageConfig,
		MetadataPath:  metadataPath,
	}
}

func NewAuthenticateSpec(
	name string,
	storageConfig *shared.StorageConfig,
	metadataPath string,
	connectorName shared.Service,
	connectorConfig auth.Config,
) Spec {
	return &AuthenticateSpec{
		BasePythonSpec: BasePythonSpec{
			BaseSpec: BaseSpec{
				Type: AuthenticateJobType,
				Name: name,
			},
			StorageConfig: *storageConfig,
			MetadataPath:  metadataPath,
		},
		ConnectorName:   connectorName,
		ConnectorConfig: connectorConfig,
	}
}

// NewExtractSpec constructs a Spec for an ExtractJob.
func NewExtractSpec(
	name string,
	storageConfig *shared.StorageConfig,
	metadataPath string,
	connectorName shared.Service,
	connectorConfig auth.Config,
	parameters connector.ExtractParams,
	inputParamNames []string,
	inputContentPaths []string,
	inputMetadataPaths []string,
	outputContentPath string,
	outputMetadataPath string,
) Spec {
	return &ExtractSpec{
		BasePythonSpec: BasePythonSpec{
			BaseSpec: BaseSpec{
				Type: ExtractJobType,
				Name: name,
			},
			StorageConfig: *storageConfig,
			MetadataPath:  metadataPath,
		},
		InputContentPaths:  inputContentPaths,
		InputMetadataPaths: inputMetadataPaths,
		ConnectorName:      connectorName,
		ConnectorConfig:    connectorConfig,
		Parameters:         parameters,
		OutputContentPath:  outputContentPath,
		OutputMetadataPath: outputMetadataPath,
	}
}

// NewDeleteSavedObjectsSpec constructs a Spec for a DeleteWrittenObjectsJob.
func NewDeleteSavedObjectsSpec(
	name string,
	storageConfig *shared.StorageConfig,
	metadataPath string,
	connectorName map[string]shared.Service,
	connectorConfig map[string]auth.Config,
	integrationToObject map[string][]string,
	outputContentPath string,
) Spec {
	return &DeleteSavedObjectsSpec{
		BasePythonSpec: BasePythonSpec{
			BaseSpec: BaseSpec{
				Type: DeleteSavedObjectsJobType,
				Name: name,
			},
			StorageConfig: *storageConfig,
			MetadataPath:  metadataPath,
		},
		ConnectorName:       connectorName,
		ConnectorConfig:     connectorConfig,
		IntegrationToObject: integrationToObject,
		OutputContentPath:   outputContentPath,
	}
}

// NewLoadTableSpec constructs a Spec for a LoadTableJob.
func NewLoadTableSpec(
	name string,
	csv string,
	storageConfig *shared.StorageConfig,
	metadataPath string,
	connectorName shared.Service,
	connectorConfig auth.Config,
	parameters connector.LoadParams,
	inputContentPath string,
	inputMetadataPath string,
) Spec {
	return &LoadTableSpec{
		BasePythonSpec: BasePythonSpec{
			BaseSpec: BaseSpec{
				Type: LoadTableJobType,
				Name: name,
			},
			StorageConfig: *storageConfig,
			MetadataPath:  metadataPath,
		},
		ConnectorName:   connectorName,
		ConnectorConfig: connectorConfig,
		CSV:             csv,
		LoadParameters: LoadSpec{
			BasePythonSpec: BasePythonSpec{
				BaseSpec: BaseSpec{
					Type: LoadJobType,
					Name: name,
				},
				StorageConfig: *storageConfig,
				MetadataPath:  metadataPath,
			},
			ConnectorName:     connectorName,
			ConnectorConfig:   connectorConfig,
			Parameters:        parameters,
			InputContentPath:  inputContentPath,
			InputMetadataPath: inputMetadataPath,
		},
	}
}

// NewDiscoverSpec constructs a Spec for a DiscoverJob.
func NewDiscoverSpec(
	name string,
	storageConfig *shared.StorageConfig,
	metadataPath string,
	connectorName shared.Service,
	connectorConfig auth.Config,
	outputContentPath string,
) Spec {
	return &DiscoverSpec{
		BasePythonSpec: BasePythonSpec{
			BaseSpec: BaseSpec{
				Type: DiscoverJobType,
				Name: name,
			},
			StorageConfig: *storageConfig,
			MetadataPath:  metadataPath,
		},
		ConnectorName:     connectorName,
		ConnectorConfig:   connectorConfig,
		OutputContentPath: outputContentPath,
	}
}

func NewCompileAirflowSpec(
	name string,
	workflowDagID uuid.UUID,
	storageConfig *shared.StorageConfig,
	metadataPath string,
	outputContentPath string,
	dagId string,
	cronSchedule string,
	taskSpecs map[string]Spec,
	taskEdges map[string][]string,
) (Spec, error) {
	for _, taskSpec := range taskSpecs {
		if taskSpec.Type() != ExtractJobType &&
			taskSpec.Type() != FunctionJobType &&
			taskSpec.Type() != ParamJobType &&
			taskSpec.Type() != SystemMetricJobType &&
			taskSpec.Type() != LoadJobType {
			return nil, errors.Newf("Task specs cannot be of type %v", taskSpec.Type())
		}
	}

	return &CompileAirflowSpec{
		BasePythonSpec: BasePythonSpec{
			BaseSpec: BaseSpec{
				Type: CompileAirflowJobType,
				Name: name,
			},
			StorageConfig: *storageConfig,
			MetadataPath:  metadataPath,
		},
		WorkflowDagId:     workflowDagID,
		OutputContentPath: outputContentPath,
		DagId:             dagId,
		CronSchedule:      cronSchedule,
		TaskSpecs:         taskSpecs,
		TaskEdges:         taskEdges,
	}, nil
}

// `EncodeSpec` first serialize `spec` according to `SerializationType` and returns the base64 encoded string.
// The encoded string can be safely passed around without any escaping issue (e.g. as envVar)
func EncodeSpec(spec Spec, serializationType SerializationType) (string, error) {
	var specData []byte
	var err error
	if serializationType == JsonSerializationType {
		specData, err = json.Marshal(spec)
		if err != nil {
			return "", err
		}
		return base64.StdEncoding.EncodeToString(specData), nil
	} else if serializationType == GobSerializationType {
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		if err := encoder.Encode(&spec); err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
	}

	return "", errors.Newf("Unsupported serialization type %s.", serializationType)
}

func DecodeSpec(specData string, serializationType SerializationType) (Spec, error) {
	specBytes, err := base64.StdEncoding.DecodeString(specData)
	if err != nil {
		return nil, err
	}

	var spec Spec
	if serializationType == JsonSerializationType {
		var base BaseSpec
		if err := json.Unmarshal(specBytes, &base); err != nil {
			return nil, err
		}

		switch base.Type {
		case WorkflowJobType:
			spec = &WorkflowSpec{}
		case WorkflowRetentionType:
			spec = &WorkflowRetentionSpec{}
		case FunctionJobType:
			spec = &FunctionSpec{}
		case AuthenticateJobType:
			spec = &AuthenticateSpec{}
		case ExtractJobType:
			spec = &ExtractSpec{}
		case LoadJobType:
			spec = &LoadSpec{}
		case LoadTableJobType:
			spec = &LoadTableSpec{}
		case DiscoverJobType:
			spec = &DiscoverSpec{}
		default:
			return nil, errors.Newf("Unknown job type: %v", base.Type)
		}

		if err := json.Unmarshal(specBytes, spec); err != nil {
			return nil, err
		}

		return spec, nil
	} else if serializationType == GobSerializationType {
		buf := bytes.NewBuffer(specBytes)
		decoder := gob.NewDecoder(buf)
		if err := decoder.Decode(&spec); err != nil {
			return nil, err
		}

		return spec, nil
	}

	return nil, errors.Newf("Unsupported serialization type %s.", serializationType)
}
