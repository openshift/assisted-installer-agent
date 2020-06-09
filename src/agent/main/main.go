package main

import (
	"fmt"

	"github.com/ori-amizur/introspector/src/commands"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/util"
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
