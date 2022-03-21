package actions

import (
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

type ntpSynchronizer struct {
	args []string
}

func (a ntpSynchronizer) Validate() error {
	modelToValidate := models.NtpSynchronizationRequest{}
	err := validateCommon("ntp synchronizer", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a ntpSynchronizer) Run() (string, []string) {
	podmanRunCmd := []string{
		"run", "--privileged", "--net=host", "--rm",
		"-v", "/usr/bin/chronyc:/usr/bin/chronyc",
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		"-v", "/var/run/chrony:/var/run/chrony",
		config.GlobalAgentConfig.AgentVersion,
		"ntp_synchronizer",
	}
	podmanRunCmd = append(podmanRunCmd, a.args...)
	return podman, podmanRunCmd
}
