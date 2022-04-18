package actions

import (
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

type diskPerfCheck struct {
	args []string
}

func (a *diskPerfCheck) Validate() error {
	modelToValidate := models.DiskSpeedCheckRequest{}
	err := validateCommon("disk performance", 2, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	if _, err := strconv.ParseFloat(a.args[1], 64); err != nil {
		log.WithError(err).Errorf("Failed to parse timeout value to float: %s", a.args[1])
		return err
	}

	return nil
}

func (a *diskPerfCheck) Command() string {
	return "sh"
}

func (a *diskPerfCheck) Args() []string {
	arguments := []string{
		"-c",
		"id=`podman ps --quiet --filter \"name=disk_performance\"` ; " +
			"test ! -z \"$id\" || " +
			fmt.Sprintf("timeout %s ", a.args[1]) +
			"podman run --privileged --rm --quiet -v /dev:/dev:rw -v /var/log:/var/log -v /run/systemd/journal/socket:/run/systemd/journal/socket " +
			"--name disk_performance " +
			config.GlobalAgentConfig.AgentVersion + " disk_speed_check '" +
			a.args[0] + "'",
	}
	return arguments
}

func (a *diskPerfCheck) Run() (stdout, stderr string, exitCode int) {
	return util.ExecutePrivileged(a.Command(), a.Args()...)
}
