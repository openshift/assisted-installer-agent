package commands

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/scanners"
	"github.com/ori-amizur/introspector/src/session"
)

var CurrentHost *models.Host

func createRegisterParams() *installer.RegisterHostParams {
	ret := &installer.RegisterHostParams{
		ClusterID: strfmt.UUID(config.GlobalAgentConfig.ClusterID),
		NewHostParams: &models.HostCreateParams{
			HostID:                scanners.ReadId(scanners.NewGHWSerialDiscovery()),
			DiscoveryAgentVersion: config.GlobalAgentConfig.AgentVersion,
		},
	}
	return ret
}

func RegisterHostWithRetry() {
	for {
		s := session.New()
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
