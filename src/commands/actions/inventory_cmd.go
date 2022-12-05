package actions

import (
	"fmt"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/config"
)

type inventory struct {
	args        []string
	agentConfig *config.AgentConfig
}

func (a *inventory) Validate() error {
	err := ValidateCommon("inventory", 1, a.args, nil)
	if err != nil {
		return err
	}
	if !strfmt.IsUUID(a.args[0]) {
		return fmt.Errorf("inventory cmd accepts only 1 params in args and it should be UUID, given args %v", a.args)
	}
	return nil
}

func (a *inventory) Run() (stdout, stderr string, exitCode int) {
	return util.ExecutePrivileged(a.Command(), a.Args()...)
}

func (a *inventory) Command() string {
	return "sh"
}

func (a *inventory) Args() []string {
	// Copying mounts file, which is not available by podman's PID
	// We incorporate the host's ID in the copied mtab file path to allow multiple agents
	// to run on the same host during load testing easily without fighting over the same
	// path (each of them has a different fake host ID)
	mtabPath := fmt.Sprintf("/root/mtab-%s", a.args[0])
	mtabCopy := fmt.Sprintf("cp /etc/mtab %s", mtabPath)
	mtabMount := fmt.Sprintf("%s:/host/etc/mtab:ro", mtabPath)

	podmanRunCmd := strings.Join([]string{
		podman, "run", "--privileged", "--pid=host", "--net=host", "--rm", "--quiet",
		"-v", "/var/log:/var/log",
		"-v", "/run/udev:/run/udev",
		"-v", "/dev/disk:/dev/disk",
		"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",

		// Enable capturing host's HW using a different root path for GHW library
		"-v", "/var/log:/host/var/log:ro",
		"-v", "/proc/meminfo:/host/proc/meminfo:ro",
		"-v", "/sys/kernel/mm/hugepages:/host/sys/kernel/mm/hugepages:ro",
		"-v", "/proc/cpuinfo:/host/proc/cpuinfo:ro",
		"-v", mtabMount,
		"-v", "/sys/block:/host/sys/block:ro",
		"-v", "/sys/devices:/host/sys/devices:ro",
		"-v", "/sys/bus:/host/sys/bus:ro",
		"-v", "/sys/class:/host/sys/class:ro",
		"-v", "/run/udev:/host/run/udev:ro",
		"-v", "/dev/disk:/host/dev/disk:ro",

		a.agentConfig.AgentVersion,
		"inventory",
	}, " ")

	return []string{"-c", fmt.Sprintf("%v && %v", mtabCopy, podmanRunCmd)}
}
