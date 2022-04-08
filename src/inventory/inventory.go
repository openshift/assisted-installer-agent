package inventory

import (
	"encoding/json"
	"time"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

func ReadInventory(subprocessConfig *config.SubprocessConfig, c *Options) *models.Inventory {
	d := util.NewDependencies(&subprocessConfig.DryRunConfig, c.GhwChrootRoot)
	ret := models.Inventory{
		BmcAddress:   GetBmcAddress(subprocessConfig, d),
		BmcV6address: GetBmcV6Address(subprocessConfig, d),
		Boot:         GetBoot(d),
		CPU:          GetCPU(d),
		Disks:        GetDisks(subprocessConfig, d),
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

func CreateInventoryInfo(subprocessConfig *config.SubprocessConfig) []byte {
	in := ReadInventory(subprocessConfig, &Options{GhwChrootRoot: "/host"})

	if subprocessConfig.DryRunEnabled {
		applyDryRunConfig(subprocessConfig, in)
	}

	b, _ := json.Marshal(&in)
	return b
}

type Options struct {
	GhwChrootRoot string
}
