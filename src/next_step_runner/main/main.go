package main

import (
	"context"
	"sync"
	"time"

	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.ProcessArgs()
	config.ProcessDryRunArgs()
	util.SetLogging("agent_next_step_runner", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging, config.GlobalDryRunConfig.ForcedHostID)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(1)
	nextStepRunnerFactory := commands.NewNextStepRunnerFactory()
	go commands.ProcessSteps(ctx, nextStepRunnerFactory, &wg)

	if config.GlobalDryRunConfig.DryRunEnabled {
		log.Info(`Dry run enabled, will cancel goroutine on fake "reboot"`)
		for {
			if util.DryRebootHappened() {
				log.Info("Dry reboot happened, exiting")
				cancel()
				break
			}

			time.Sleep(time.Second)
		}
	} else {
		// Nothing interesting to do, wait for the goroutine to finish naturally
		wg.Wait()
	}

	log.Info("next step runner exiting")
}
