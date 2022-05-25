package actions

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/pkg/errors"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/config"
)

type inventory struct {
	args        []string
	agentConfig *config.AgentConfig
}

func (a *inventory) Validate() error {
	if len(a.args) < 1 || len(a.args) > 2 {
		return fmt.Errorf("%s cmd accepts 1 - 2 params in args, given args %v", "inventory", a.args)
	}
	if !strfmt.IsUUID(a.args[0]) {
		return fmt.Errorf("inventory cmd accepts only a UUID as the first arg, given args %v", a.args)
	}
	if len(a.args) == 2 {
		if _, err := strconv.ParseBool(a.args[1]); err != nil {
			return errors.Wrapf(err, "inventory cmd only accepts a boolean variable as the second arg, given args %v", a.args)
		}
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

	podmanRunCmd := []string{
		podman, "run", "--privileged", "--net=host", "--rm", "--quiet",
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
	}
	if len(a.args) > 1 {
		podmanRunCmd = append(podmanRunCmd, fmt.Sprintf("--enable-virtual-interfaces=%s", a.args[1]))
	}
	return []string{"-c", fmt.Sprintf("%v && %v", mtabCopy, strings.Join(podmanRunCmd, " "))}
}
