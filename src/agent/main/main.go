package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/agent"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/disk_speed_check"
	"github.com/openshift/assisted-installer-agent/src/free_addresses"
	"github.com/openshift/assisted-installer-agent/src/inventory"
	"github.com/openshift/assisted-installer-agent/src/logs_sender"
	"github.com/openshift/assisted-installer-agent/src/next_step_runner"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) == 0 {
		logrus.Fatal("No arguments were passed to the agent")
	}

	// All binaries are actually just symlinks to the same agent binary, so we
	// need to check the binary name to know which binary we're supposed to
	// behave as
	binaryName := filepath.Base(os.Args[0])
	binaries := map[string]func(){
		"agent":            Main,
		"free_addresses":   free_addresses.Main,
		"inventory":        inventory.Main,
		"logs_sender":      logs_sender.Main,
		"next_step_runner": next_step_runner.Main,
		"disk_speed_check": disk_speed_check.Main,
	}
	if mainFunc, ok := binaries[binaryName]; ok {
		mainFunc()
	} else {
		logrus.Fatalf("unknown binary name %s, expected one of %s", binaryName, strings.Join(getMapKeysSorted(binaries), ", "))
	}
}

func getMapKeysSorted(binaries map[string]func()) []string {
	keys := make([]string, 0, len(binaries))
	for k := range binaries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func Main() {
	agentConfig := config.ProcessArgs()
	config.ProcessDryRunArgs(&agentConfig.DryRunConfig)
	util.SetLogging("agent_registration", agentConfig.TextLogging, agentConfig.JournalLogging, agentConfig.StdoutLogging, agentConfig.ForcedHostID)
	nextStepRunnerFactory := agent.NewNextStepRunnerFactory()
	agent.RunAgent(agentConfig, nextStepRunnerFactory, logrus.StandardLogger())
}
