package util

import "github.com/openshift/assisted-installer-agent/src/config"

func DryRebootHappened() bool {
	// The dry run installer creates this file on "Reboot" (instead of actually rebooting)
	// We use this as a signal that we should terminate as well
	if _, _, exitCode := ExecutePrivileged("stat", config.GlobalDryRunConfig.FakeRebootMarkerPath); exitCode == 0 {
		return true
	}

	return false
}
