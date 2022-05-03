package actions

import (
	"fmt"
	"strconv"

	"github.com/openshift/assisted-installer-agent/src/config"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type nextStepRunner struct {
	args                 []string
	nextStepRunnerParams models.NextStepCmdRequest
}

func (a *nextStepRunner) Validate() error {
	err := validateCommon("next step runner", 1, a.args, &a.nextStepRunnerParams)
	if err != nil {
		return err
	}
	return nil
}

func (a *nextStepRunner) CreateCmd() (string, []string) {
	arguments := []string{"run", "--rm", "-ti", "--privileged", "--pid=host", "--net=host",
		"-v", "/dev:/dev:rw", "-v", "/opt:/opt:rw",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		"-v", "/var/log:/var/log:rw",
		"-v", "/run/media:/run/media:rw",
		"-v", "/etc/pki:/etc/pki"}

	if config.GlobalAgentConfig.CACertificatePath != "" {
		arguments = append(arguments, "-v", fmt.Sprintf("%s:%s", config.GlobalAgentConfig.CACertificatePath,
			config.GlobalAgentConfig.CACertificatePath))
	}

	arguments = append(arguments,
		"--env", "PULL_SECRET_TOKEN",
		"--env", "CONTAINERS_CONF",
		"--env", "CONTAINERS_STORAGE_CONF",
		"--env", "HTTP_PROXY", "--env", "HTTPS_PROXY", "--env", "NO_PROXY",
		"--env", "http_proxy", "--env", "https_proxy", "--env", "no_proxy",
		"--name", "next-step-runner", swag.StringValue(a.nextStepRunnerParams.AgentVersion), "next_step_runner",
		"--url", config.GlobalAgentConfig.TargetURL,
		"--infra-env-id", a.nextStepRunnerParams.InfraEnvID.String(),
		"--host-id", a.nextStepRunnerParams.HostID.String(),
		"--agent-version", swag.StringValue(a.nextStepRunnerParams.AgentVersion),
		fmt.Sprintf("--insecure=%s", strconv.FormatBool(config.GlobalAgentConfig.InsecureConnection)))
	if config.GlobalAgentConfig.CACertificatePath != "" {
		arguments = append(arguments, "--cacert", config.GlobalAgentConfig.CACertificatePath)
	}

	return podman, arguments
}

func StartStepRunner(command string, args []string) error {
	log.Infof("Running next step runner. Command: %s, Args: %s", command, args)
	if command == "" {
		runner := nextStepRunner{args: args}
		err := runner.Validate()
		if err != nil {
			log.WithError(err).Errorf("next step runner command validation failed")
			return err
		}
		command, args = runner.CreateCmd()
	}

	_, stderr, exitCode := util.Execute(command, args...)
	if exitCode != 0 {
		return errors.Errorf("next step runner command exited with non-zero exit code %d: %s", exitCode, stderr)
	}
	return nil
}