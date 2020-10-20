package commands

import (
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func GetInventory(string, ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute("podman", "run", "--privileged", "--net=host", "--rm", "-v", "/var/log:/var/log",
		"-v", "/run/udev:/run/udev", "-v", "/dev/disk:/dev/disk", "-v", "/run/systemd/journal/socket:/run/systemd/journal/socket", config.GlobalAgentConfig.InventoryImage, "inventory")
}
