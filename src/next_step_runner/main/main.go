package main

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/openshift/assisted-installer-agent/pkg/shutdown"
	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Create a context:
	ctx := context.Background()

	agentConfig := config.ProcessArgs()
	config.ProcessDryRunArgs(&agentConfig.DryRunConfig)
	util.SetLogging("agent_next_step_runner", agentConfig.TextLogging, agentConfig.JournalLogging, agentConfig.ForcedHostID)

	// Prepare the shutdown sequence:
	shutdown, err := shutdown.NewSequence().
		Logger(log.StandardLogger()).
		Signal(syscall.SIGKILL).
		Signal(syscall.SIGTERM).
		Build()
	if err != nil {
		log.WithError(err).Error("Failed to create shutdown sequence")
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	toolRunnerFactory := commands.NewToolRunnerFactory()
	go commands.ProcessSteps(ctx, agentConfig, toolRunnerFactory, &wg, log.StandardLogger(), shutdown)

	if agentConfig.DryRunEnabled {
		log.Info(`Dry run enabled, will cancel goroutine on fake "reboot"`)
		for {
			if util.DryRebootHappened(&agentConfig.DryRunConfig) {
				log.Info("Dry reboot happened, exiting")
				shutdown.Start(0)
				break
			}

			time.Sleep(time.Second)
		}
	} else {
		// Nothing interesting to do, wait for the goroutine to finish naturally
		wg.Wait()
	}

	// Start the shutdown sequence:
	shutdown.Start(0)
}
