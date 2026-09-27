package actions

import (
	"github.com/openshift/assisted-installer-agent/src/dhcp_lease_allocate"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type dhcpLeases struct {
	args []string
}

func (a *dhcpLeases) Validate() error {
	modelToValidate := models.DhcpAllocationRequest{}
	err := ValidateCommon("dhcp leases", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *dhcpLeases) Run() (stdout, stderr string, exitCode int) {
	leaser := dhcp_lease_allocate.NewLeaser(dhcp_lease_allocate.NewLeaserDependencies())
	return leaser.LeaseAllocate(a.args[0], log.StandardLogger())
}

func (a *dhcpLeases) Command() string {
	return "dhcp_leases"
}

func (a *dhcpLeases) Args() []string {
	return a.args
}
