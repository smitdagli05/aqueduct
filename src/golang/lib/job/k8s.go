package job

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aqueducthq/aqueduct/lib"
	"github.com/aqueducthq/aqueduct/lib/k8s"
	"github.com/aqueducthq/aqueduct/lib/models/shared"
	"github.com/aqueducthq/aqueduct/lib/models/shared/operator"
	"github.com/aqueducthq/aqueduct/lib/models/shared/operator/function"
	"github.com/dropbox/godropbox/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	jobSpecEnvVarKey = "JOB_SPEC"
)

type k8sJobManager struct {
	// When we initialize k8sJobManager, k8sClient is always set to nil. This is because
	// in case of dynamic k8s integration, when we initialize the job manager, the k8s
	// cluster may not exist yet, so k8s client creation will fail. We defer the initialization
	// to Launch and Poll, at which point regardless of dynamic or static k8s integration, we expect
	// the k8s client creation to succeed.
	k8sClient *kubernetes.Clientset
	conf      *K8sJobManagerConfig
}

func setupNamespaceAndSecrets(k8sClient *kubernetes.Clientset, conf *K8sJobManagerConfig) error {
	err := k8s.CreateNamespaces(k8sClient)
	if err != nil {
		return errors.Wrap(err, "Error while creating K8s Namespaces")
	}

	secretsMap := map[string]string{}
	secretsMap[k8s.AwsAccessKeyIdName] = conf.AwsAccessKeyId
	secretsMap[k8s.AwsAccessKeyName] = conf.AwsSecretAccessKey
	err = k8s.CreateSecret(context.Background(), k8s.AwsCredentialsSecretName, secretsMap, k8sClient)
	if err != nil {
		// Double-check that we didn't race against another process to create this secret.
		if _, secretExistsErr := k8s.GetSecret(context.Background(), k8s.AwsCredentialsSecretName, k8sClient); secretExistsErr != nil {
			return errors.Wrap(err, "Error while creating K8s Secrets")
		}
	}

	return nil
}

func (j *k8sJobManager) initialize() error {
	k8sClient, err := k8s.CreateK8sClient(j.conf.KubeconfigPath, j.conf.UseSameCluster)
	if err != nil {
		return errors.Wrap(err, "Error while creating K8sClient")
	}

	err = setupNamespaceAndSecrets(k8sClient, j.conf)
	if err != nil {
		return err
	}

	j.k8sClient = k8sClient

	return nil
}

func NewK8sJobManager(conf *K8sJobManagerConfig) (*k8sJobManager, error) {
	return &k8sJobManager{
		k8sClient: nil,
		conf:      conf,
	}, nil
}

func (j *k8sJobManager) Config() Config {
	return j.conf
}

func (j *k8sJobManager) Launch(ctx context.Context, name string, spec Spec) JobError {
	if j.k8sClient == nil {
		if err := j.initialize(); err != nil {
			return systemError(err)
		}
	}

	launchGpu := false
	var cudaVersion operator.CudaVersionNumber
	resourceRequest := map[string]string{
		k8s.PodResourceCPUKey:    k8s.DefaultCPURequest,
		k8s.PodResourceMemoryKey: k8s.DefaultMemoryRequest,
	}

	environmentVariables := map[string]string{}

	if spec.Type() == FunctionJobType {
		functionSpec, ok := spec.(*FunctionSpec)
		if !ok {
			return systemError(errors.Newf("Function Spec is expected, but got %v", spec))
		}

		functionSpec.FunctionExtractPath = defaultFunctionExtractPath

		if functionSpec.Resources != nil {
			if functionSpec.Resources.GPUResourceName != nil {
				resourceRequest[k8s.GPUResourceName] = *functionSpec.Resources.GPUResourceName
				launchGpu = true
			}

			if functionSpec.Resources.CudaVersion != nil {
				cudaVersion = *functionSpec.Resources.CudaVersion
			} else {
				cudaVersion = k8s.DefaultCudaVersion
			}

			if functionSpec.Resources.NumCPU != nil {
				resourceRequest[k8s.PodResourceCPUKey] = strconv.Itoa(*functionSpec.Resources.NumCPU)
			}
			if functionSpec.Resources.MemoryMB != nil {
				// Set the request to be in "M" = Megabytes.
				resourceRequest[k8s.PodResourceMemoryKey] = fmt.Sprintf("%sM",
					strconv.Itoa(*functionSpec.Resources.MemoryMB),
				)
			}
		}
	}

	// Encode job spec to prevent data loss
	serializationType := JsonSerializationType
	encodedSpec, err := EncodeSpec(spec, serializationType)
	if err != nil {
		return systemError(err)
	}

	environmentVariables[jobSpecEnvVarKey] = encodedSpec

	secretEnvVars := []string{}

	if spec.HasStorageConfig() {
		// This job spec has a storage config that k8s needs access to
		storageConfig, err := spec.GetStorageConfig()
		if err != nil {
			return systemError(err)
		}

		if storageConfig.Type == shared.S3StorageType {
			// k8s clusters access S3 via credentials passed as a secret
			secretEnvVars = append(secretEnvVars, k8s.AwsCredentialsSecretName)
		}
	}

	containerRepo, err := mapJobTypeToDockerImage(spec, launchGpu, cudaVersion)
	if err != nil {
		return userError(err)
	}
	containerImage := fmt.Sprintf("%s:%s", containerRepo, lib.ServerVersionNumber)

	err = k8s.LaunchJob(
		name,
		containerImage,
		&environmentVariables,
		secretEnvVars,
		&resourceRequest,
		j.k8sClient,
	)
	if err != nil {
		return systemError(err)
	}
	return nil
}

func containerStatusFromPod(pod *corev1.Pod, name string) (*corev1.ContainerStatus, error) {
	if len(pod.Status.ContainerStatuses) != 1 {
		return nil, errors.Newf(
			"Expected job %s to have one container, but instead got %v.",
			name,
			len(pod.Status.ContainerStatuses),
		)
	}

	containerStatus := pod.Status.ContainerStatuses[0]
	if containerStatus.State.Terminated == nil {
		return nil, errors.Newf(
			"Container %s should have terminated.", containerStatus.Name,
		)
	}
	return &containerStatus, nil
}

func (j *k8sJobManager) Poll(ctx context.Context, name string) (shared.ExecutionStatus, JobError) {
	if j.k8sClient == nil {
		if err := j.initialize(); err != nil {
			return shared.UnknownExecutionStatus, systemError(err)
		}
	}

	job, err := k8s.GetJob(ctx, name, j.k8sClient)
	if err != nil {
		return shared.UnknownExecutionStatus, jobMissingError(err)
	}

	var status shared.ExecutionStatus
	if job.Status.Succeeded == 1 {
		status = shared.SucceededExecutionStatus
	} else if job.Status.Failed == 1 {
		status = shared.FailedExecutionStatus

		// Fetch more detailed information about the failure, in case there is valuable
		// context we can surface to the user.
		pod, err := k8s.GetPod(ctx, name, j.k8sClient)
		if err != nil {
			return status, systemError(err)
		}

		containerStatus, err := containerStatusFromPod(pod, name)
		if err != nil {
			return status, systemError(err)
		}

		if containerStatus.State.Terminated.Reason == "OOMKilled" {
			return status, userError(errors.New("Operator failed on Kubernetes due to Out-of-Memory exception."))
		}

		// We do not error here since pods are killed with a failing exit status on any failed checks.
		// We should rely on the written execution state to decide whether to continue dag execution,
		// and not the status of the pod.
		return status, nil
	} else {
		_, err := k8s.GetPod(ctx, name, j.k8sClient)
		if err != nil {
			if err == k8s.ErrNoPodExists {
				return shared.PendingExecutionStatus, nil
			}
			return shared.FailedExecutionStatus, systemError(err)
		}

		status = shared.PendingExecutionStatus
	}

	return status, nil
}

func (j *k8sJobManager) DeployCronJob(ctx context.Context, name string, period string, spec Spec) JobError {
	return nil
}

func (j *k8sJobManager) CronJobExists(ctx context.Context, name string) bool {
	return false
}

func (j *k8sJobManager) EditCronJob(ctx context.Context, name string, cronString string) JobError {
	return nil
}

func (j *k8sJobManager) DeleteCronJob(ctx context.Context, name string) JobError {
	return nil
}

// Maps a job Spec to Docker image.
func mapJobTypeToDockerImage(spec Spec, launchGpu bool, cudaVersion operator.CudaVersionNumber) (string, error) {
	switch spec.Type() {
	case FunctionJobType:
		functionSpec, ok := spec.(*FunctionSpec)
		if !ok {
			return "", errors.New("Unable to determine Python Version.")
		}
		pythonVersion, err := function.GetPythonVersion(context.TODO(), functionSpec.FunctionPath, &functionSpec.StorageConfig)
		if err != nil {
			return "", errors.New("Unable to determine Python Version.")
		}
		if launchGpu {
			return mapGpuFunctionToDockerImage(pythonVersion, cudaVersion)
		} else {
			switch pythonVersion {
			case function.PythonVersion37:
				return Function37DockerImage, nil
			case function.PythonVersion38:
				return Function38DockerImage, nil
			case function.PythonVersion39:
				return Function39DockerImage, nil
			case function.PythonVersion310:
				return Function310DockerImage, nil
			default:
				return "", errors.New("Unable to determine Python Version.")
			}
		}

	case AuthenticateJobType:
		authenticateSpec := spec.(*AuthenticateSpec)
		return mapIntegrationServiceToDockerImage(authenticateSpec.ConnectorName)
	case ExtractJobType:
		extractSpec := spec.(*ExtractSpec)
		return mapIntegrationServiceToDockerImage(extractSpec.ConnectorName)
	case LoadJobType:
		loadSpec := spec.(*LoadSpec)
		return mapIntegrationServiceToDockerImage(loadSpec.ConnectorName)
	case DiscoverJobType:
		discoverSpec := spec.(*DiscoverSpec)
		return mapIntegrationServiceToDockerImage(discoverSpec.ConnectorName)
	case ParamJobType:
		return ParameterDockerImage, nil
	case SystemMetricJobType:
		return SystemMetricDockerImage, nil
	default:
		return "", errors.Newf("Unsupported job type %v provided", spec.Type())
	}
}

func mapIntegrationServiceToDockerImage(service shared.Service) (string, error) {
	switch service {
	case shared.Postgres, shared.Redshift, shared.AqueductDemo:
		return PostgresConnectorDockerImage, nil
	case shared.Snowflake:
		return SnowflakeConnectorDockerImage, nil
	case shared.MySql, shared.MariaDb:
		return MySqlConnectorDockerImage, nil
	case shared.SqlServer:
		return SqlServerConnectorDockerImage, nil
	case shared.BigQuery:
		return BigQueryConnectorDockerImage, nil
	case shared.S3:
		return S3ConnectorDockerImage, nil
	default:
		return "", errors.Newf("Unknown integration service provided %v", service)
	}
}

func mapGpuFunctionToDockerImage(pythonVersion function.PythonVersion, cudaVersion operator.CudaVersionNumber) (string, error) {
	switch cudaVersion {
	case operator.Cuda11_8_0:
		switch pythonVersion {
		case function.PythonVersion37:
			return GpuCuda1180Python37, nil
		case function.PythonVersion38:
			return GpuCuda1180Python38, nil
		case function.PythonVersion39:
			return GpuCuda1180Python39, nil
		case function.PythonVersion310:
			return GpuCuda1180Python310, nil
		default:
			return "", errors.New("Unable to determine Python Version.")
		}
	case operator.Cuda11_4_1:
		switch pythonVersion {
		case function.PythonVersion37:
			return GpuCuda1141Python37, nil
		case function.PythonVersion38:
			return GpuCuda1141Python38, nil
		case function.PythonVersion39:
			return GpuCuda1141Python39, nil
		case function.PythonVersion310:
			return GpuCuda1141Python310, nil
		default:
			return "", errors.New("Unable to determine Python Version.")
		}
	default:
		return "", errors.New("Unsupported CUDA version provided. We currently only support CUDA versions 11.4.1 and 11.8.0")
	}
}
