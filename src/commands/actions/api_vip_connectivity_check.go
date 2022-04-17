package actions

import (
	"github.com/openshift/assisted-installer-agent/src/apivip_check"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

type apiVipConnectivityCheck struct {
	args []string
}

func (a *apiVipConnectivityCheck) Validate() error {
	modelToValidate := models.APIVipConnectivityRequest{}
	err := ValidateCommon("api vip connectivity check", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *apiVipConnectivityCheck) Run() (stdout, stderr string, exitCode int) {
	return apivip_check.CheckAPIConnectivity(a.args[0], logrus.StandardLogger())
}

func (a *apiVipConnectivityCheck) Command() string {
	return "api_vip_connectivity_check"
}

func (a *apiVipConnectivityCheck) Args() []string {
	return a.args
}
