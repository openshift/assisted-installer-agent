package actions

import (
	"github.com/openshift/assisted-installer-agent/src/upgrade_agent"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type upgradeAgent struct {
	args []string
}

func (u *upgradeAgent) Validate() error {
	var request models.UpgradeAgentRequest
	return ValidateCommon("upgrade agent", 1, u.args, &request)
}

func (u *upgradeAgent) Run() (stdout, stderr string, exitCode int) {
	return upgrade_agent.Run(u.Args()[0], &upgrade_agent.RealDependencies{}, log.StandardLogger())
}

func (u *upgradeAgent) Command() string {
	return "upgrade_agent"
}

func (u *upgradeAgent) Args() []string {
	return u.args
}
