package commands

import (
	"context"
	"github.com/go-openapi/strfmt"
	"time"

	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/client"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/scanners"
	log "github.com/sirupsen/logrus"
)

var CurrentHost *models.Host

func createRegisterParams() *inventory.RegisterHostParams {
	ret := &inventory.RegisterHostParams{
		ClusterID: strfmt.UUID(config.GlobalConfig.ClusterID),
		NewHostParams: &models.HostCreateParams{
			HostID:    scanners.ReadId(),
		},
	}
	return ret
}

func RegisterHostWithRetry() {
	bmInventory := client.CreateBmInventoryClient()
	for {
		registerResult, err := bmInventory.Inventory.RegisterHost(context.Background(), createRegisterParams())
		if err == nil {
			log.Infof("Payload is %+v", registerResult.Payload)
			CurrentHost = registerResult.Payload
			return
		}
		log.Warnf("Error registering host: %s", err.Error())
		time.Sleep(time.Duration(config.GlobalConfig.IntervalSecs) * time.Second)
	}
}
