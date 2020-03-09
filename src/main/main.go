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
	} else {
		commands.RegisterNodeWithRetry()
	}
}
