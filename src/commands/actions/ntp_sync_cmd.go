package actions

import (
	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/ntp_synchronizer"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/assisted-service/pkg/validations"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ntpSynchronizer struct {
	args        []string
	agentConfig *config.AgentConfig
}

func (a *ntpSynchronizer) Validate() error {
	var request models.NtpSynchronizationRequest
	err := ValidateCommon("ntp synchronizer", 1, a.args, &request)
	if err != nil {
		return err
	}

	ntpSource := swag.StringValue(request.NtpSource)
	if ntpSource != "" && !validations.ValidateAdditionalNTPSource(ntpSource) {
		err = errors.Errorf("Invalid NTP source: %s", ntpSource)
		return err
	}

	return nil
}

func (a *ntpSynchronizer) Run() (stdout, stderr string, exitCode int) {
	return ntp_synchronizer.Run(a.Args()[0], &ntp_synchronizer.ProcessExecuter{}, log.StandardLogger())
}

func (a *ntpSynchronizer) Command() string {
	return "ntp_synchronizer"
}

func (a *ntpSynchronizer) Args() []string {
	return a.args
}
