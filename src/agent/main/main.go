package main

import (
	"fmt"

	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	config.ProcessArgs()
	util.SetLogging("agent", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging)
	if config.GlobalAgentConfig.IsText {
		o, _, _ := commands.GetInventory("")
		fmt.Print(o)
	} else if config.GlobalAgentConfig.ConnectivityParams != "" {
		output, errStr, exitCode := commands.ConnectivityCheck("", config.GlobalAgentConfig.ConnectivityParams)
		if exitCode != 0 {
			fmt.Println(errStr)
		} else {
			fmt.Println(output)
		}
	} else {
		commands.RegisterHostWithRetry()
		commands.ProcessSteps()
	}
}
