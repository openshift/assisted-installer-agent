package actions

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strconv"

	"github.com/openshift/assisted-installer-agent/src/util"

	"github.com/openshift/assisted-installer-agent/src/config"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-service/models"
)

const containerName = "next-step-runner"

type nextStepRunnerAction struct {
	args                 []string
	nextStepRunnerParams models.NextStepCmdRequest
	agentConfig          *config.AgentConfig
}

func NewNextStepRunnerAction(agentConfig *config.AgentConfig, args []string) ActionInterface {
	return &nextStepRunnerAction{args: args, agentConfig: agentConfig}
}

func (a *nextStepRunnerAction) Validate() error {
	err := ValidateCommon("next step runner", 1, a.args, &a.nextStepRunnerParams)
	if err != nil {
		return err
	}
	return nil
}

func (a *nextStepRunnerAction) Run() (stdout, stderr string, exitCode int) {
	return util.ExecutePrivileged(a.Command(), a.Args()...)
}

func (a *nextStepRunnerAction) Command() string {
	return podman
}

// Try to cleanup previous next-step-runner if it exists, best effort
func (a *nextStepRunnerAction) cleanupPrevious() {
	_, stderr, exitCode := util.ExecutePrivileged(a.Command(), "rm", "-i", containerName)
	if exitCode != 0 {
		log.Warnf("Failed to cleanup old %s container, stdErr %s, exitCode %d.", containerName, stderr, exitCode)
	}
}

func (a *nextStepRunnerAction) Args() []string {
	a.cleanupPrevious()

	arguments := []string{"run", "--rm", "-ti", "--privileged", "--pid=host", "--net=host",

		// unlimited number of processes in the container
		"--pids-limit=0",
		"-v", "/dev:/dev:rw", "-v", "/opt:/opt:rw",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		"-v", "/var/log:/var/log:rw",
		"-v", "/run/media:/run/media:rw",
		"-v", "/usr/bin/chronyc:/usr/bin/chronyc",
		"-v", "/var/run/chrony:/var/run/chrony",
		"-v", "/etc/pki:/etc/pki"}

	if a.agentConfig.CACertificatePath != "" {
		arguments = append(arguments, "-v", fmt.Sprintf("%s:%s", a.agentConfig.CACertificatePath,
			a.agentConfig.CACertificatePath))
	}

	arguments = append(arguments,
		"--env", "PULL_SECRET_TOKEN",
		"--env", "CONTAINERS_CONF",
		"--env", "CONTAINERS_STORAGE_CONF",
		"--env", "HTTP_PROXY", "--env", "HTTPS_PROXY", "--env", "NO_PROXY",
		"--env", "http_proxy", "--env", "https_proxy", "--env", "no_proxy",
		"--name", containerName, swag.StringValue(a.nextStepRunnerParams.AgentVersion), "next_step_runner",
		"--url", a.agentConfig.TargetURL,
		"--infra-env-id", a.nextStepRunnerParams.InfraEnvID.String(),
		"--host-id", a.nextStepRunnerParams.HostID.String(),
		"--agent-version", swag.StringValue(a.nextStepRunnerParams.AgentVersion),
		fmt.Sprintf("--insecure=%s", strconv.FormatBool(a.agentConfig.InsecureConnection)))

	if a.agentConfig.CACertificatePath != "" {
		arguments = append(arguments, "--cacert", a.agentConfig.CACertificatePath)
	}

	return arguments
}
