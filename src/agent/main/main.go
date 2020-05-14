package main

import (
	"fmt"
	"github.com/ori-amizur/introspector/src/util"

	"github.com/ori-amizur/introspector/src/commands"
	"github.com/ori-amizur/introspector/src/config"
)

func main() {
	util.SetLogging("agent")
	config.ProcessArgs()
	if config.GlobalConfig.IsText {
		o, _, _ := commands.GetInventory("")
		fmt.Print(o)
	} else if config.GlobalConfig.ConnectivityParams != "" {
		output, errStr, exitCode := commands.ConnectivityCheck("", config.GlobalConfig.ConnectivityParams)
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
