package main

import (
	"fmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/inventory"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	config.ProcessDryRunArgs()
	config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	util.SetLogging("inventory", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging, config.GlobalDryRunConfig.ForcedHostID)
	fmt.Print(string(inventory.CreateInventoryInfo()))
}
