package actions

import (
	"net/http"

	"github.com/openshift/assisted-installer-agent/src/tang_connectivity_check"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

type tangConnectivityCheck struct {
	args []string
}

func (a *tangConnectivityCheck) Validate() error {
	modelToValidate := models.TangConnectivityRequest{}
	err := ValidateCommon("tang connectivity check", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *tangConnectivityCheck) Run() (stdout, stderr string, exitCode int) {
	client := &http.Client{}
	return tang_connectivity_check.CheckTangConnectivity(a.args[0], logrus.StandardLogger(), client)
}

func (a *tangConnectivityCheck) Command() string {
	return "tang_connectivity_check"
}

func (a *tangConnectivityCheck) Args() []string {
	return a.args
}
