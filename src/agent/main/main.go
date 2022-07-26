package main

import (
	"github.com/openshift/assisted-installer-agent/src/agent"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/sirupsen/logrus"
)

func main() {
	agentConfig := config.ProcessArgs()
	config.ProcessDryRunArgs(&agentConfig.DryRunConfig)
	util.SetLogging("agent_registration", agentConfig.TextLogging, agentConfig.JournalLogging, agentConfig.StdoutLogging, agentConfig.ForcedHostID)
	nextStepRunnerFactory := agent.NewNextStepRunnerFactory()
	agent.RunAgent(agentConfig, nextStepRunnerFactory, logrus.StandardLogger())
}
