package actions

import (
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

type domainResolution struct {
	args []string
}

func (a *domainResolution) Validate() error {
	modelToValidate := models.DomainResolutionRequest{}
	err := validateCommon("domain resolution", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *domainResolution) Run() (string, []string) {
	podmanRunCmd := []string{
		"run", "--privileged", "--net=host", "--rm", "--quiet",
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		config.GlobalAgentConfig.AgentVersion,
		"domain_resolution",
		"-request",
		a.args[0],
	}

	return podman, podmanRunCmd
}
