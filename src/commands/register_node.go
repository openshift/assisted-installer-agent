package commands

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/session"
	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"
)

func RegisterHostWithRetry() *models.HostRegistrationResponseAO1NextStepRunnerCommand {

	for {
		s, err := session.New(config.GlobalAgentConfig.TargetURL, config.GlobalAgentConfig.PullSecretToken)
		if err != nil {
			logrus.Fatalf("Failed to initialize connection: %e", err)
		}
		serviceAPI := newServiceAPI()

		registerResult, err := serviceAPI.RegisterHost(s)
		if err == nil {
			return registerResult.NextStepRunnerCommand
		}

		// stop register in case of forbidden reply.
		switch err.(type) {
		case *installer.V2RegisterHostForbidden:
			s.Logger().Warn("Host will stop trying to register; host is not allowed to perform the requested operation")
			// wait forever
			select {}
		case *installer.V2RegisterHostConflict:
			s.Logger().Warn("Host will stop trying to register; cluster cannot accept new hosts in its current state")
			// wait forever
			select {}
		case *installer.V2RegisterHostNotFound:
			s.Logger().Warnf("Host will stop trying to register; infra-env id %s does not exist, or user is not authorized", config.GlobalAgentConfig.InfraEnvID)
			// wait forever
			select {}
		case *installer.V2RegisterHostUnauthorized:
			s.Logger().Warnf("Host will stop trying to register; user is not authenticated to perform host registration")
			// wait forever
			select {}
		default:
			s.Logger().Warnf("Error registering host: %s", getErrorMessage(err))
		}
		time.Sleep(time.Duration(config.GlobalAgentConfig.IntervalSecs) * time.Second)
	}
}
