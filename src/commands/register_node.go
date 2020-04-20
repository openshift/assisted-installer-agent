package commands

import (
	"github.com/go-openapi/strfmt"
	"github.com/ori-amizur/introspector/src/util"
	"time"

	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/client"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/scanners"
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
		ctx := client.NewContext()
		registerResult, err := bmInventory.Inventory.RegisterHost(ctx, createRegisterParams())
		if err == nil {
			CurrentHost = registerResult.Payload
			return
		}
		util.ToLogger(ctx).Warnf("Error registering host: %s", err.Error())
		time.Sleep(time.Duration(config.GlobalConfig.IntervalSecs) * time.Second)
	}
}
