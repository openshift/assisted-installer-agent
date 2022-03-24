package actions

import (
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

type apiVipConnectivityCheck struct {
	args []string
}

func (a *apiVipConnectivityCheck) Validate() error {
	modelToValidate := models.APIVipConnectivityRequest{}
	err := validateCommon("api vip connectivity check", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *apiVipConnectivityCheck) CreateCmd() (string, []string) {
	podmanRunCmd := []string{
		"run", "--privileged", "--net=host", "--rm", "--quiet",
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		config.GlobalAgentConfig.AgentVersion,
		"apivip_check",
	}

	return podman, append(podmanRunCmd, a.args...)
}
