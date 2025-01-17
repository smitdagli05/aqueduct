package job

import (
	"context"
	"encoding/gob"
	"path"

	"github.com/aqueducthq/aqueduct/lib/lib_utils"
	"github.com/aqueducthq/aqueduct/lib/models/shared"
	"github.com/aqueducthq/aqueduct/lib/vault"
	"github.com/aqueducthq/aqueduct/lib/workflow/operator/connector/auth"
	"github.com/dropbox/godropbox/errors"
)

type ManagerType string

const (
	ProcessType    ManagerType = "process"
	K8sType        ManagerType = "k8s"
	LambdaType     ManagerType = "lambda"
	DatabricksType ManagerType = "databricks"
	SparkType      ManagerType = "spark"
)

type Config interface {
	Type() ManagerType
}

type ProcessConfig struct {
	BinaryDir             string `yaml:"binaryDir" json:"binary_dir"`
	LogsDir               string `yaml:"logsDir" json:"logs_dir"`
	PythonExecutorPackage string `yaml:"pythonExecutorPackage" json:"python_executor_package"`
	OperatorStorageDir    string `yaml:"operatorStorageDir" json:"operator_storage_dir"`
	CondaEnvName          string `yaml:"condaEnvName" json:"conda_env_name"`
}

type K8sJobManagerConfig struct {
	KubeconfigPath     string `yaml:"kubeconfigPath" json:"kubeconfig_path"`
	ClusterName        string `yaml:"clusterName" json:"cluster_name"`
	UseSameCluster     bool   `json:"use_same_cluster"  yaml:"useSameCluster"`
	AwsAccessKeyId     string `yaml:"awsAccessKeyId" json:"aws_access_key_id"`
	AwsSecretAccessKey string `yaml:"awsSecretAccessKey" json:"aws_secret_access_key"`

	// System config, will have defaults
	AwsRegion string `yaml:"awsRegion" json:"aws_region"`

	Dynamic bool `yaml:"dynamic" json:"dynamic"`
}

type LambdaJobManagerConfig struct {
	RoleArn            string `yaml:"roleArn" json:"role_arn"`
	AwsAccessKeyId     string `yaml:"awsAccessKeyId" json:"aws_access_key_id"`
	AwsSecretAccessKey string `yaml:"awsSecretAccessKey" json:"aws_secret_access_key"`
}

type DatabricksJobManagerConfig struct {
	// WorkspaceURL is the full url for the Databricks workspace that
	// Aqueduct operators will run on.
	WorkspaceURL string `yaml:"workspaceUrl" json:"workspace_url"`
	// AccessToken is a Databricks AccessToken for a workspace. Information on how
	// to create tokens can be found here: https://docs.databricks.com/dev-tools/auth.html#personal-access-tokens-for-users
	AccessToken string `yaml:"accessToken" json:"access_token"`
	// Databricks needs an Instance Profile with S3 permissions in order to access metadata
	// storage in S3. Information on how to create this can be found here:
	// https://docs.databricks.com/aws/iam/instance-profile-tutorial.html
	S3InstanceProfileARN string `yaml:"s3InstanceProfileArn" json:"s3_instance_profile_arn"`
	// [Optional] ID of instance pool that Aqueduct-created JobClusters should use.
	InstancePoolID *string `yaml:"instancePoolID" json:"instance_pool_id"`
	// AWS Access Key ID is passed from the StorageConfig.
	AwsAccessKeyID string `yaml:"awsAccessKeyId" json:"aws_access_key_id"`
	// AWS Secret Access Key is passed from the StorageConfig.
	AwsSecretAccessKey string `yaml:"awsSecretAccessKey" json:"aws_secret_access_key"`
}

type SparkJobManagerConfig struct {
	// LivyServerURL is the URL of the Livy server that sits in front of the Spark cluster.
	// This URL is assumed to be accessible by the machine running the Aqueduct server.
	LivyServerURL string `yaml:"baseUrl" json:"livy_server_url"`
	// AWS Access Key ID is passed from the StorageConfig.
	AwsAccessKeyID string `yaml:"awsAccessKeyId" json:"aws_access_key_id"`
	// AWS Secret Access Key is passed from the StorageConfig.
	AwsSecretAccessKey string `yaml:"awsSecretAccessKey" json:"aws_secret_access_key"`
	// URI to the packaged environment. This is passed when creating and uploading the
	// environment during execution.
	EnvironmentPathURI string `yaml:"environmentPathUri" json:"environment_path_uri"`
}

func (*ProcessConfig) Type() ManagerType {
	return ProcessType
}

func (*K8sJobManagerConfig) Type() ManagerType {
	return K8sType
}

func (*LambdaJobManagerConfig) Type() ManagerType {
	return LambdaType
}

func (*DatabricksJobManagerConfig) Type() ManagerType {
	return DatabricksType
}

func (*SparkJobManagerConfig) Type() ManagerType {
	return SparkType
}

func RegisterGobTypes() {
	gob.Register(&ProcessConfig{})
	gob.Register(&K8sJobManagerConfig{})
	gob.Register(&WorkflowSpec{})
	gob.Register(&WorkflowRetentionSpec{})
	gob.Register(&DynamicTeardownSpec{})
}

func init() {
	RegisterGobTypes()
}

func GenerateNewJobManager(
	ctx context.Context,
	engineConfig shared.EngineConfig,
	storageConfig *shared.StorageConfig,
	aqPath string,
	vault vault.Vault,
) (JobManager, error) {
	jobConfig, err := GenerateJobManagerConfig(
		ctx,
		engineConfig,
		storageConfig,
		aqPath,
		vault,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to generate JobManagerConfig.")
	}

	jobManager, err := NewJobManager(jobConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create JobManager.")
	}
	return jobManager, nil
}

func GenerateJobManagerConfig(
	ctx context.Context,
	engineConfig shared.EngineConfig,
	storageConfig *shared.StorageConfig,
	aqPath string,
	vault vault.Vault,
) (Config, error) {
	switch engineConfig.Type {
	case shared.AqueductEngineType:
		return &ProcessConfig{
			BinaryDir:          path.Join(aqPath, BinaryDir),
			OperatorStorageDir: path.Join(aqPath, OperatorStorageDir),
		}, nil
	case shared.AqueductCondaEngineType:
		return &ProcessConfig{
			BinaryDir:          path.Join(aqPath, BinaryDir),
			OperatorStorageDir: path.Join(aqPath, OperatorStorageDir),
			CondaEnvName:       engineConfig.AqueductCondaConfig.Env,
		}, nil
	case shared.K8sEngineType:
		if storageConfig.Type != shared.S3StorageType && storageConfig.Type != shared.GCSStorageType {
			return nil, errors.New("Must use S3 or GCS storage config for K8s engine.")
		}

		var awsAccessKeyId, awsSecretAccessKey string
		if storageConfig.Type == shared.S3StorageType {
			keyId, secretKey, err := lib_utils.ExtractAwsCredentials(storageConfig.S3Config)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to extract AWS credentials from file.")
			}

			awsAccessKeyId = keyId
			awsSecretAccessKey = secretKey
		}

		k8sIntegrationId := engineConfig.K8sConfig.IntegrationID
		config, err := auth.ReadConfigFromSecret(ctx, k8sIntegrationId, vault)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to read k8s config from vault.")
		}
		k8sConfig, err := lib_utils.ParseK8sConfig(config)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to parse k8s config.")
		}

		return &K8sJobManagerConfig{
			KubeconfigPath:     k8sConfig.KubeconfigPath,
			ClusterName:        k8sConfig.ClusterName,
			UseSameCluster:     bool(k8sConfig.UseSameCluster),
			AwsAccessKeyId:     awsAccessKeyId,
			AwsSecretAccessKey: awsSecretAccessKey,
			Dynamic:            bool(k8sConfig.Dynamic),
		}, nil
	case shared.LambdaEngineType:
		if storageConfig.Type != shared.S3StorageType {
			return nil, errors.New("Must use S3 for Lambda engine.")
		}
		lambdaIntegrationId := engineConfig.LambdaConfig.IntegrationID
		config, err := auth.ReadConfigFromSecret(ctx, lambdaIntegrationId, vault)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to read config from vault.")
		}
		lambdaConfig, err := lib_utils.ParseLambdaConfig(config)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get integration.")
		}

		var awsAccessKeyId, awsSecretAccessKey string
		if storageConfig.Type == shared.S3StorageType {
			keyId, secretKey, err := lib_utils.ExtractAwsCredentials(storageConfig.S3Config)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to extract AWS credentials from file.")
			}

			awsAccessKeyId = keyId
			awsSecretAccessKey = secretKey
		}

		return &LambdaJobManagerConfig{
			RoleArn:            lambdaConfig.RoleArn,
			AwsAccessKeyId:     awsAccessKeyId,
			AwsSecretAccessKey: awsSecretAccessKey,
		}, nil
	case shared.DatabricksEngineType:
		if storageConfig.Type != shared.S3StorageType {
			return nil, errors.New("Must use S3 storage config for Databricks engine.")
		}
		databricksIntegrationId := engineConfig.DatabricksConfig.IntegrationID
		config, err := auth.ReadConfigFromSecret(ctx, databricksIntegrationId, vault)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to read config from vault.")
		}
		databricksConfig, err := lib_utils.ParseDatabricksConfig(config)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get integration.")
		}

		var awsAccessKeyId, awsSecretAccessKey string
		if storageConfig.Type == shared.S3StorageType {
			keyId, secretKey, err := lib_utils.ExtractAwsCredentials(storageConfig.S3Config)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to extract AWS credentials from file.")
			}

			awsAccessKeyId = keyId
			awsSecretAccessKey = secretKey
		}
		return &DatabricksJobManagerConfig{
			WorkspaceURL:         databricksConfig.WorkspaceURL,
			AccessToken:          databricksConfig.AccessToken,
			S3InstanceProfileARN: databricksConfig.S3InstanceProfileARN,
			InstancePoolID:       databricksConfig.InstancePoolID,
			AwsAccessKeyID:       awsAccessKeyId,
			AwsSecretAccessKey:   awsSecretAccessKey,
		}, nil
	case shared.SparkEngineType:
		if storageConfig.Type != shared.S3StorageType {
			return nil, errors.New("Must use S3 storage config for Databricks engine.")
		}
		sparkIntegrationID := engineConfig.SparkConfig.IntegrationId
		config, err := auth.ReadConfigFromSecret(ctx, sparkIntegrationID, vault)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to read config from vault.")
		}
		sparkConfig, err := lib_utils.ParseSparkConfig(config)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get integration.")
		}

		var awsAccessKeyId, awsSecretAccessKey string
		if storageConfig.Type == shared.S3StorageType {
			keyId, secretKey, err := lib_utils.ExtractAwsCredentials(storageConfig.S3Config)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to extract AWS credentials from file.")
			}

			awsAccessKeyId = keyId
			awsSecretAccessKey = secretKey
		}
		return &SparkJobManagerConfig{
			LivyServerURL:      sparkConfig.LivyServerURL,
			AwsAccessKeyID:     awsAccessKeyId,
			AwsSecretAccessKey: awsSecretAccessKey,
			EnvironmentPathURI: engineConfig.SparkConfig.EnvironmentPathURI,
		}, nil
	default:
		return nil, errors.New("Unsupported engine type.")
	}
}
