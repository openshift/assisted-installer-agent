package actions

import (
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

type connectivityCheck struct {
	args []string
}

func (a *connectivityCheck) Validate() error {
	modelToValidate := models.ConnectivityCheckParams{}
	err := validateCommon("connectivity check", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *connectivityCheck) Run() (string, []string) {
	commandArgs := []string{
		"run", "--privileged", "--net=host", "--rm", "--quiet",
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		config.GlobalAgentConfig.AgentVersion,
		"connectivity_check",
	}
	return podman, append(commandArgs, a.args...)
}
