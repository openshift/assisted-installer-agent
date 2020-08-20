package commands

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/go-openapi/strfmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/scanners"
	"github.com/openshift/assisted-installer-agent/src/session"
	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"
)

var CurrentHost *models.Host

func createRegisterParams() *installer.RegisterHostParams {
	ret := &installer.RegisterHostParams{
		ClusterID: strfmt.UUID(config.GlobalAgentConfig.ClusterID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.AgentVersion,
		NewHostParams: &models.HostCreateParams{
			HostID:                scanners.ReadId(scanners.NewGHWSerialDiscovery()),
			DiscoveryAgentVersion: config.GlobalAgentConfig.AgentVersion,
		},
	}
	return ret
}

func RegisterHostWithRetry() {
	for {
		s, err := session.New(config.GlobalAgentConfig.TargetURL, config.GlobalAgentConfig.PullSecretToken)
		if err != nil {
			logrus.Fatalf("Failed to initialize connection: %e", err)
		}
		registerResult, err := s.Client().Installer.RegisterHost(s.Context(), createRegisterParams())
		if err == nil {
			CurrentHost = registerResult.Payload
			return
		}
		// stop register in case of forbidden reply.
		switch err.(type) {
		case *installer.RegisterHostForbidden, *installer.RegisterHostNotFound:
			s.Logger().Warn("Host will stop trying to register")
			// wait forever
			select {}
		}
		s.Logger().Warnf("Error registering host: %s", err.Error())
		time.Sleep(time.Duration(config.GlobalAgentConfig.IntervalSecs) * time.Second)
	}
}
