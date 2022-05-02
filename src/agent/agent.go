package agent

import (
	"time"

	"github.com/openshift/assisted-installer-agent/src/commands"
	"github.com/openshift/assisted-installer-agent/src/commands/actions"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

const defaultRetryDelay = 1 * time.Hour

type nextStepRunnerFactory struct{}

func NewNextStepRunnerFactory() commands.NextStepRunnerFactory {
	return &nextStepRunnerFactory{}
}

func (n *nextStepRunnerFactory) Create(agentConfig *config.AgentConfig, args []string) (commands.Runner, error) {
	action := actions.NewNextStepRunnerAction(agentConfig, args)
	err := action.Validate()
	if err != nil {
		log.WithError(err).Errorf("next step runner command validation failed")
		return nil, err
	}
	return action, nil
}

func delayOnError(stepRunnerCommand *models.HostRegistrationResponseAO1NextStepRunnerCommand) time.Duration {
	if stepRunnerCommand.RetrySeconds > 0 {
		return time.Duration(stepRunnerCommand.RetrySeconds) * time.Second
	} else {
		return defaultRetryDelay
	}
}

func RunAgent(agentConfig *config.AgentConfig, nextStepRunnerFactory commands.NextStepRunnerFactory, log log.FieldLogger) {
	for {
		stepRunnerCommand := commands.RegisterHostWithRetry(agentConfig, log)
		if stepRunnerCommand == nil {
			log.Errorf("Incompatible server version, going to retry in %s", defaultRetryDelay)
			time.Sleep(defaultRetryDelay)
			continue
		}

		nextStepRunner, err := nextStepRunnerFactory.Create(agentConfig, stepRunnerCommand.Args)
		if err != nil {
			reRegisterDelay := delayOnError(stepRunnerCommand)
			log.WithError(err).Errorf("Unable to create next step runner. Attempt again in %s", reRegisterDelay)
			time.Sleep(reRegisterDelay)
			continue
		}

		log.Infof("Running next step runner. Command: %s, Args: %s", nextStepRunner.Command(), nextStepRunner.Args())
		_, stderr, exitCode := nextStepRunner.Run()
		if exitCode != 0 {
			reRegisterDelay := delayOnError(stepRunnerCommand)
			log.WithField("stderr", stderr).
				WithField("exitCode", exitCode).
				Errorf("Next step runner has crashed and will be restarted in %s", reRegisterDelay)
			time.Sleep(reRegisterDelay)
			continue
		}

		if agentConfig.DryRunEnabled {
			// Check if the step runner died just because the installer signaled fake reboot
			if util.DryRebootHappened(&agentConfig.DryRunConfig) {
				log.Infof("Dry reboot happened, exiting")
				break
			}
		}

		log.Info("Next step runner exited, going to re-register host")
	}
}
