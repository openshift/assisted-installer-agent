package main

import (
	"time"

	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

const defaultRetryDelay = 1 * time.Hour

func main() {
	config.ProcessArgs()
	config.ProcessDryRunArgs()
	util.SetLogging("agent_registration", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging, config.GlobalDryRunConfig.ForcedHostID)

	for {
		stepRunnerCommand := commands.RegisterHostWithRetry()
		if stepRunnerCommand == nil {
			log.Errorf("Incompatible server version, going to retry in %s", defaultRetryDelay)
			time.Sleep(defaultRetryDelay)
			continue
		}

		if err := commands.StartStepRunner(stepRunnerCommand.Command, stepRunnerCommand.Args...); err != nil {
			var reRegistrerDelay time.Duration
			if stepRunnerCommand.RetrySeconds > 0 {
				reRegistrerDelay = time.Duration(stepRunnerCommand.RetrySeconds) * time.Second
			} else {
				reRegistrerDelay = defaultRetryDelay
			}

			if config.GlobalDryRunConfig.DryRunEnabled {
				// Check if the step runner died just because the installer signaled fake reboot
				if util.DryRebootHappened() {
					log.Infof("Dry reboot happened, exiting")
					return
				}
			}

			log.WithError(err).Errorf("Next step runner has crashed and will be restarted in %s", reRegistrerDelay)
			time.Sleep(reRegistrerDelay)
			continue
		}

		log.Info("Next step runner exited, going to re-register host")
	}
}
