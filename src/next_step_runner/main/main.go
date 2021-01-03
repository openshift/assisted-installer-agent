package main

import (
	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.ProcessArgs()
	log.Warnf("%v", config.GlobalAgentConfig)
	util.SetLogging("agent_next_step_runner", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging)
	commands.ProcessSteps()
	log.Info("next step runner exiting")
}
