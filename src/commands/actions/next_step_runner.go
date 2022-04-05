package actions

import (
	"fmt"
	"strconv"

	"github.com/openshift/assisted-installer-agent/src/util"

	"github.com/openshift/assisted-installer-agent/src/config"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-service/models"
)

type nextStepRunnerAction struct {
	args                 []string
	nextStepRunnerParams models.NextStepCmdRequest
}

func NewNextStepRunnerAction(args []string) ActionInterface {
	return &nextStepRunnerAction{args: args}
}

func (a *nextStepRunnerAction) Validate() error {
	err := validateCommon("next step runner", 1, a.args, &a.nextStepRunnerParams)
	if err != nil {
		return err
	}
	return nil
}

func (a *nextStepRunnerAction) CreateCmd() (string, []string) {
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

	return a.Command(), arguments
}

func (a *nextStepRunnerAction) Run() (stdout, stderr string, exitCode int) {
	command, args := a.CreateCmd()
	return util.ExecutePrivileged(command, args...)
}

func (a *nextStepRunnerAction) Command() string {
	return podman
}

func (a *nextStepRunnerAction) Args() []string {
	return a.args
}
