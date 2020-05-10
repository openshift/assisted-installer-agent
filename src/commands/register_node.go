package commands

import (
	"github.com/go-openapi/strfmt"
	"time"

	"github.com/filanov/bm-inventory/client/installer"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/scanners"
	"github.com/ori-amizur/introspector/src/session"
)

var CurrentHost *models.Host

func createRegisterParams() *installer.RegisterHostParams {
	ret := &installer.RegisterHostParams{
		ClusterID: strfmt.UUID(config.GlobalConfig.ClusterID),
		NewHostParams: &models.HostCreateParams{
			HostID:    scanners.ReadId(),
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
		s.Logger().Warnf("Error registering host: %s", err.Error())
		time.Sleep(time.Duration(config.GlobalConfig.IntervalSecs) * time.Second)
	}
}
