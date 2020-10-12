package main

import (
	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	config.ProcessArgs()
	util.SetLogging("agent_next_step_runner", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging)
	commands.ProcessSteps()
}
