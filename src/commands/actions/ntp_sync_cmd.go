package actions

import (
	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/assisted-service/pkg/validations"
	"github.com/pkg/errors"
)

type ntpSynchronizer struct {
	args        []string
	agentConfig *config.AgentConfig
}

func (a *ntpSynchronizer) Validate() error {
	var request models.NtpSynchronizationRequest
	err := validateCommon("ntp synchronizer", 1, a.args, &request)
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
	return util.ExecutePrivileged(a.Command(), a.Args()...)
}

func (a *ntpSynchronizer) Command() string {
	return podman
}

func (a *ntpSynchronizer) Args() []string {
	podmanRunCmd := []string{
		"run", "--privileged", "--net=host", "--rm",
		"-v", "/usr/bin/chronyc:/usr/bin/chronyc",
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		"-v", "/var/run/chrony:/var/run/chrony",
		a.agentConfig.AgentVersion,
		"ntp_synchronizer",
	}
	podmanRunCmd = append(podmanRunCmd, a.args...)
	return podmanRunCmd
}
