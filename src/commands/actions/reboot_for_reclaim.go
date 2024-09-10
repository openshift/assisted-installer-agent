package actions

import (
	"encoding/json"
	"fmt"
	"runtime"
	"syscall"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

type rebootForReclaim struct {
	args []string
}

func (a *rebootForReclaim) Validate() error {
	return ValidateCommon("reboot for reclaim", 1, a.args, &models.RebootForReclaimRequest{})
}

func (a *rebootForReclaim) Run() (stdout, stderr string, exitCode int) {
	var req models.RebootForReclaimRequest
	if err := json.Unmarshal([]byte(a.args[0]), &req); err != nil {
		return "", fmt.Sprintf("failed unmarshalling reboot for reclaim request: %s", err.Error()), -1
	}

	if err := syscall.Chroot(*req.HostFsMountDir); err != nil {
		return "", err.Error(), -1
	}

	if runtime.GOARCH == "s390x" {
		unshareCommand := "unshare"
		unshareArgs := []string{
			"--mount",
			"bash",
			"-c",
			fmt.Sprintf("mount -o remount,rw /boot && zipl -V -t /boot -i %s/%s -r %s/%s -c /boot/loader/entries%s -p /boot/loader/entries%s",
				artifactsFolder, kernelFile,
				artifactsFolder, initrdFile,
				bootLoaderConfigFileName,
				bootLoaderConfigFileName),
		}
		stdout, stderr, exitCode = util.Execute(unshareCommand, unshareArgs...)
		if exitCode != 0 {
			return stdout, stderr, exitCode
		}
	}
	return util.Execute("systemctl", "reboot")
}

// Unused, but required as part of ActionInterface
func (a *rebootForReclaim) Command() string {
	return "systemctl"
}

// Unused, but required as part of ActionInterface
func (a *rebootForReclaim) Args() []string {
	return []string{"reboot"}
}
