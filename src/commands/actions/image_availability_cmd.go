package actions

import (
	"fmt"

	"github.com/openshift/assisted-installer-agent/src/util"

	"github.com/alessio/shellescape"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

type imageAvailability struct {
	args []string
}

func (a *imageAvailability) Validate() error {
	modelToValidate := models.ContainerImageAvailabilityRequest{}
	err := validateCommon("image availability", 1, a.args, &modelToValidate)
	if err != nil {
		return err
	}
	return nil
}

func (a *imageAvailability) CreateCmd() (string, []string) {
	const containerName = "container_image_availability"

	podmanRunCmd := shellescape.QuoteCommand([]string{
		"podman", "run", "--privileged", "--net=host", "--rm", "--quiet", "--pid=host",
		"--name", containerName,
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		config.GlobalAgentConfig.AgentVersion,
		"container_image_availability",
		"--request", a.args[0],
	})

	// checking if it exists and only running if it doesn't
	checkAlreadyRunningCmd := fmt.Sprintf("podman ps --format '{{.Names}}' | grep -q '^%s$'", containerName)

	return a.Command(), []string{"-c", fmt.Sprintf("%s || %s", checkAlreadyRunningCmd, podmanRunCmd)}
}

func (a *imageAvailability) Run() (stdout, stderr string, exitCode int) {
	command, args := a.CreateCmd()
	return util.ExecutePrivileged(command, args...)
}

func (a *imageAvailability) Command() string {
	return "sh"
}

func (a *imageAvailability) Args() []string {
	return a.args
}