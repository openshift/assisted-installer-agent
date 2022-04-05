package actions

import (
	"fmt"

	"github.com/alessio/shellescape"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

type freeAddresses struct {
	args []string
}

func (a *freeAddresses) Validate() error {
	modelToValidate := models.FreeAddressesRequest{}
	err := validateCommon("free addresses", 1, a.args, &modelToValidate)
	return err
}

func (a *freeAddresses) CreateCmd() (string, []string) {
	const containerName = "free_addresses_scanner"
	podmanRunCmd := []string{
		podman, "run", "--privileged", "--net=host", "--rm", "--quiet",
		"--name", containerName,
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		config.GlobalAgentConfig.AgentVersion,
		"free_addresses",
	}

	podmanRunCmd = append(podmanRunCmd, a.args...)
	cmdString := shellescape.QuoteCommand(podmanRunCmd)

	// Sometimes the address scanning takes longer than the interval we wait between invocations.
	// To avoid flooding the log with "container already exists" errors, we silently fail by manually
	// checking if it exists and only running if it doesn't
	checkAlreadyRunningCmd := fmt.Sprintf("podman ps --format '{{.Names}}' | grep -q '^%s$'", containerName)

	return a.Command(), []string{"-c", fmt.Sprintf("%s || %s", checkAlreadyRunningCmd, cmdString)}
}

func (a *freeAddresses) Run() (stdout, stderr string, exitCode int) {
	command, args := a.CreateCmd()
	return util.ExecutePrivileged(command, args...)
}

func (a *freeAddresses) Command() string {
	return "sh"
}

func (a *freeAddresses) Args() []string {
	return a.args
}