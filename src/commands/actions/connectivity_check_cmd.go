package actions

import (
	"github.com/openshift/assisted-installer-agent/src/connectivity_check"
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

func (a *connectivityCheck) Command() string {
	return "connectivity_check"
}

func (a *connectivityCheck) Args() []string {
	return a.args
}

func (a *connectivityCheck) Run() (stdout, stderr string, exitCode int) {
	return connectivity_check.ConnectivityCheck("", a.args...)
}
