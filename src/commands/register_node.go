package commands

import (
	"net/http"
	"time"

	"github.com/go-openapi/swag"

	"github.com/sirupsen/logrus"

	"github.com/go-openapi/strfmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/scanners"
	"github.com/openshift/assisted-installer-agent/src/session"
	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"
)

var CurrentHostID *strfmt.UUID

func createRegisterParams() *installer.RegisterHostParams {
	ret := &installer.RegisterHostParams{
		ClusterID:             strfmt.UUID(config.GlobalAgentConfig.ClusterID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.AgentVersion,
		NewHostParams: &models.HostCreateParams{
			HostID:                CurrentHostID,
			DiscoveryAgentVersion: config.GlobalAgentConfig.AgentVersion,
		},
	}
	return ret
}

func RegisterHostWithRetry() {
	for {
		var err error
		s, err := session.New(config.GlobalAgentConfig.TargetURL, config.GlobalAgentConfig.PullSecretToken)
		if err != nil {
			logrus.Fatalf("Failed to initialize connection: %e", err)
		}

		CurrentHostID = scanners.ReadId(scanners.NewGHWSerialDiscovery())
		_, err = s.Client().Installer.RegisterHost(s.Context(), createRegisterParams())
		if err == nil {
			return
		}
		// stop register in case of forbidden reply.
		switch errValue := err.(type) {
		case *installer.RegisterHostForbidden:
			s.Logger().Warn("Host will stop trying to register; cluster cannot accept new hosts in its current state")
			// wait forever
			select {}
		case *installer.RegisterHostNotFound:
			s.Logger().Warnf("Host will stop trying to register; cluster id %s does not exist, or user is not authorized", config.GlobalAgentConfig.ClusterID)
			// wait forever
			select {}
		case *installer.RegisterHostUnauthorized:
			s.Logger().Warnf("Host will stop trying to register; user is not authenticated to perform host registration")
			// wait forever
			select {}
		case *installer.RegisterHostInternalServerError:
			s.Logger().Warnf("Error registering host: %s, %s", http.StatusText(http.StatusInternalServerError), swag.StringValue(errValue.Payload.Reason))
		case *installer.RegisterHostBadRequest:
			s.Logger().Warnf("Error registering host: %s, %s", http.StatusText(http.StatusBadRequest), swag.StringValue(errValue.Payload.Reason))
		default:
			s.Logger().WithError(err).Warn("Error registering host")
		}
		time.Sleep(time.Duration(config.GlobalAgentConfig.IntervalSecs) * time.Second)
	}
}
