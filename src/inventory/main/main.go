package main

import (
	"flag"
	"fmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/inventory"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	// Additional command-line flag to enable collecting virtual interfaces is added here since config.ProcessSubprocessArgs calls flag.Parse
	var collectVirtualInterfaces bool
	flag.BoolVar(&collectVirtualInterfaces, "enable-virtual-interfaces", false, "Enable virtual interfaces to be passed to the assisted service")

	subprocessConfig := config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	config.ProcessDryRunArgs(&subprocessConfig.DryRunConfig)
	util.SetLogging("inventory", subprocessConfig.TextLogging, subprocessConfig.JournalLogging, subprocessConfig.ForcedHostID)
	fmt.Print(string(inventory.CreateInventoryInfo(subprocessConfig, collectVirtualInterfaces)))
}
