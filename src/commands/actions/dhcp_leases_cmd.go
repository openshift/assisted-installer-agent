package actions

import (
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

type dhcpLeases struct {
	args []string
}

func (a *dhcpLeases) Validate() error {
	modelToValidate := models.DhcpAllocationRequest{}
	err := validateCommon("dhcp leases", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *dhcpLeases) Run() (string, []string) {
	podmanRunCmd := []string{
		"run", "--privileged", "--net=host", "--rm", "--quiet",
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		config.GlobalAgentConfig.AgentVersion,
		"dhcp_lease_allocate",
	}

	podmanRunCmd = append(podmanRunCmd, a.args...)
	return podman, podmanRunCmd
}
