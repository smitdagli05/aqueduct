package execution_environment

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/aqueducthq/aqueduct/lib"
	"github.com/aqueducthq/aqueduct/lib/lib_utils"
	log "github.com/sirupsen/logrus"
)

var pythonVersions = [...]string{
	"3.7",
	"3.8",
	"3.9",
	"3.10",
}

func baseEnvNameByVersion(pythonVersion string) string {
	return fmt.Sprintf("%s%s", aqueductPythonBaseEnvNamePrefix, pythonVersion)
}

// createBaseEnvs creates base python environments.
func createBaseEnvs() error {
	for _, pythonVersion := range pythonVersions {
		envName := baseEnvNameByVersion(pythonVersion)
		args := []string{
			"create",
			"-n",
			envName,
			fmt.Sprintf("python==%s", pythonVersion),
			"-y",
		}
		_, _, err := lib_utils.RunCmd(CondaCmdPrefix, args, "", false)
		if err != nil {
			return err
		}

		args = []string{
			"run",
			"-n",
			envName,
			"pip3",
			"install",
			fmt.Sprintf("aqueduct-ml==%s", lib.ServerVersionNumber),
		}
		_, _, err = lib_utils.RunCmd(CondaCmdPrefix, args, "", false)
		if err != nil {
			return err
		}
	}

	return nil
}

func ListCondaEnvs() (map[string]bool, error) {
	listArgs := []string{
		"env",
		"list",
		"--json",
	}

	stdout, _, err := lib_utils.RunCmd(CondaCmdPrefix, listArgs, "", false)
	if err != nil {
		return nil, err
	}

	type listEnvResult struct {
		Envs []string `json:"envs"`
	}

	var envs listEnvResult
	err = json.Unmarshal([]byte(stdout), &envs)
	if err != nil {
		return nil, err
	}

	results := make(map[string]bool, len(envs.Envs))
	for _, env := range envs.Envs {
		envName := path.Base(env)

		// only include aq envs and exclude base envs.
		if strings.HasPrefix(envName, aqueductEnvNamePrefix) && !strings.HasPrefix(envName, aqueductPythonBaseEnvNamePrefix) {
			results[envName] = true
		}
	}

	return results, nil
}

// `CreateCondaEnvIfNotExists` creates an conda env corresponding to
// an ExecEnv `e`'s python version and dependencies.
// It only creates the new env if it doesn't exist, otherwise the step is skipped
// assuming the existing env already matches all required dependencies.
func CreateCondaEnvIfNotExists(
	e *ExecutionEnvironment,
	condaPath string,
	existingEnvs map[string]bool,
) error {
	if _, ok := existingEnvs[e.Name()]; ok {
		return nil
	}

	// First, we create a conda env with the env object's Python version.
	createArgs := []string{
		"create",
		"-n",
		e.Name(),
		fmt.Sprintf("python==%s", e.PythonVersion),
		"-y",
	}

	_, _, err := lib_utils.RunCmd(CondaCmdPrefix, createArgs, "", false)
	if err != nil {
		return err
	}

	forkEnvPath := fmt.Sprintf(
		"%s/envs/aqueduct_python%s/lib/python%s/site-packages",
		condaPath,
		e.PythonVersion,
		e.PythonVersion,
	)
	forkArgs := []string{
		"develop",
		"-n",
		e.Name(),
		forkEnvPath,
	}

	_, _, err = lib_utils.RunCmd(CondaCmdPrefix, forkArgs, "", false)
	if err != nil {
		return err
	}

	// Then, we use pip3 to install dependencies inside this new Conda env.
	if len(e.Dependencies) > 0 {
		installArgs := append([]string{
			"run",
			"-n",
			e.Name(),
			"pip3",
			"install",
		}, e.Dependencies...)

		_, _, err = lib_utils.RunCmd(CondaCmdPrefix, installArgs, "", false)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteCondaEnv(name string) error {
	args := []string{
		"env",
		"remove",
		"-n",
		name,
	}

	_, _, err := lib_utils.RunCmd(CondaCmdPrefix, args, "", false)
	return err
}

func DeleteCondaEnv(e *ExecutionEnvironment) error {
	return deleteCondaEnv(e.Name())
}

func DeleteBaseEnvs() error {
	for _, pythonVersion := range pythonVersions {
		err := deleteCondaEnv(baseEnvNameByVersion(pythonVersion))
		if err != nil {
			return err
		}
	}

	return nil
}

// Best-effort to delete all envs and log any error
func DeleteCondaEnvs(envs []ExecutionEnvironment) {
	for _, env := range envs {
		err := DeleteCondaEnv(&env)
		if err != nil {
			log.Errorf("Failed to delete env %s: %v", env.ID.String(), err)
		}
	}
}
