package main

import (
	"time"

	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

const defaultRetryDelay = time.Duration(1 * time.Hour)

func main() {
	config.ProcessArgs()
	util.SetLogging("agent_registration", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging)

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

			log.WithError(err).Errorf("Failed to start next step runner, going to retry in %s", reRegistrerDelay)
			time.Sleep(reRegistrerDelay)
			continue
		}

		log.Info("Next step runner exited, going to re-register host")
	}
}
