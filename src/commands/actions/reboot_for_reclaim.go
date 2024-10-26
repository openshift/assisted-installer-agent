package actions

import (
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"
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
		var options string
		cmdline_params := make(map[string]string)
		var requiredCmdline string

		stdout, stderr, exitCode = util.Execute("cat", "/boot/loader/entries/00-assisted-discovery.conf")
		if exitCode != 0 {
			return stdout, stderr, exitCode
		}
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "options") {
				options = strings.TrimSpace(strings.TrimPrefix(line, "options"))
				break
			}
		}
		stdout, stderr, exitCode := util.Execute("cat", "/proc/cmdline")
		if exitCode != 0 {
			return stdout, stderr, exitCode
		}

		// Parameters to extract from cmdline for agents
		paramsToExtract := []string{
			"ip",
			"nameserver",
			"rd.znet",
			"zfcp.allow_lun_scan",
			"rd.zfcp",
			"rd.dasd",
		}

		for _, param := range paramsToExtract {
			regex := regexp.MustCompile(fmt.Sprintf(`\b%s=([^\s]+)`, param))
			match := regex.FindStringSubmatch(stdout)
			if len(match) > 1 {
				cmdline_params[param] = match[1]
			}
		}

		for key, value := range cmdline_params {
			requiredCmdline += fmt.Sprintf("%s=%s ", key, value)
		}

		unshareCommand := "unshare"
		unshareArgs := []string{
			"--mount",
			"bash",
			"-c",
			fmt.Sprintf("mount -o remount,rw /boot && zipl -V -t /boot -i %s/%s -r %s/%s -P '%s %s'",
				artifactsFolder, kernelFile,
				artifactsFolder, initrdFile,
				options,
				requiredCmdline),
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
