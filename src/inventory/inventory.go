package inventory

import (
	"encoding/json"
	"time"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

func ReadInventory(c *Options) *models.Inventory {
	d := util.NewDependencies(c.GhwChrootRoot)
	ret := models.Inventory{
		BmcAddress:   GetBmcAddress(d),
		BmcV6address: GetBmcV6Address(d),
		Boot:         GetBoot(d),
		CPU:          GetCPU(d),
		Disks:        GetDisks(d),
		Gpus:         GetGPUs(d),
		Hostname:     GetHostname(d),
		Interfaces:   GetInterfaces(d),
		Memory:       GetMemory(d),
		SystemVendor: GetVendor(d),
		Timestamp:    time.Now().Unix(),
		Routes:       GetRoutes(d),
		TpmVersion:   GetTPM(d),
	}
	return &ret
}

func CreateInventoryInfo() []byte {
	in := ReadInventory(&Options{GhwChrootRoot: "/host"})

	if config.GlobalDryRunConfig.DryRunEnabled {
		in.Interfaces[0].MacAddress = config.GlobalDryRunConfig.ForcedMacAddress
	}

	b, _ := json.Marshal(&in)
	return b
}

type Options struct {
	GhwChrootRoot string
}
