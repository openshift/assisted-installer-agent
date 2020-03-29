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

const (
	RETRY_SLEEP_SECS = 60
)

var CurrentHost *models.Host

func createRegisterParams() *inventory.RegisterHostParams {
	clusterID := strfmt.UUID(config.GlobalConfig.ClusterID)
	ret := &inventory.RegisterHostParams{
		NewHostParams: &models.HostCreateParams{
			ClusterID: &clusterID,
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
			CurrentHost = registerResult.Payload
			return
		}
		log.Warnf("Error registering host: %s", err.Error())
		time.Sleep(RETRY_SLEEP_SECS * time.Second)
	}
}
