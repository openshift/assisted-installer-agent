package main

import (
	"fmt"

	"github.com/ori-amizur/introspector/src/commands"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/util"
)

func main() {
	config.ProcessSubprocessArgs(true, false)
	util.SetLogging("connectivity-check", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging)
	fmt.Print(string(commands.CreateHostInfo()))
}
