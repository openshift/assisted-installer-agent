package commands

import (
	"context"
	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/client"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/scanners"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	RETRY_SLEEP_SECS = 60
)

var CurrentNode *models.Node

func createRegisterParams() *inventory.RegisterNodeParams {
	nodeInfo := string(CreateNodeInfo())
	ret := &inventory.RegisterNodeParams{
		NewNodeParams: &models.NodeCreateParams{
			HardwareInfo: &nodeInfo,
			Namespace:    &config.GlobalConfig.Namespace,
			Serial:       scanners.ReadMotherboadSerial(),
		},
	}
	return ret
}


func RegisterNodeWithRetry() {
	bmInventory := client.CreateBmInventoryClient()
	for {
		registerResult, err := bmInventory.Inventory.RegisterNode(context.Background(), createRegisterParams())
		if err == nil {
			CurrentNode = registerResult.Payload
			return
		}
		log.Warnf("Error registering node: %s", err.Error())
		time.Sleep(RETRY_SLEEP_SECS * time.Second)
	}
}
