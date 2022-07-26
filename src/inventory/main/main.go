package main

import (
	"fmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/inventory"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	subprocessConfig := config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	config.ProcessDryRunArgs(&subprocessConfig.DryRunConfig)
	util.SetLogging("inventory", subprocessConfig.TextLogging, subprocessConfig.JournalLogging, subprocessConfig.StdoutLogging, subprocessConfig.ForcedHostID)
	fmt.Print(string(inventory.CreateInventoryInfo(subprocessConfig)))
}
