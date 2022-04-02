package main

import (
	"github.com/openshift/assisted-installer-agent/src/agent"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	config.ProcessArgs()
	config.ProcessDryRunArgs()
	util.SetLogging("agent_registration", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging, config.GlobalDryRunConfig.ForcedHostID)
	nextStepRunnerFactory := agent.NewNextStepRunnerFactory()
	agent.RunAgent(nextStepRunnerFactory)
}
