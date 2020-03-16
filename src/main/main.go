package main

import (
	"fmt"
	"github.com/ori-amizur/introspector/src/commands"
	"github.com/ori-amizur/introspector/src/config"
)


func main() {
	config.ProcessArgs()
	if config.GlobalConfig.IsText {
		fmt.Printf("%s\n", string(commands.CreateNodeInfo()))
	} else if config.GlobalConfig.ConnectivityParams != "" {
		output, err := commands.ConnectivityCheck(config.GlobalConfig.ConnectivityParams)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(output)
		}
	} else {
		commands.RegisterNodeWithRetry()
		commands.ProcessSteps()
	}
}
