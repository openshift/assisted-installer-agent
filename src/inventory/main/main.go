package main

import (
	"fmt"

	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/inventory"
	"github.com/ori-amizur/introspector/src/util"
)

func main() {
	config.ProcessSubprocessArgs(false, true)
	util.SetLogging("inventory", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging)
	fmt.Print(string(inventory.CreateInveroryInfo()))
}
