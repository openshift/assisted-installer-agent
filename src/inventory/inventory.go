package inventory

import (
	"encoding/json"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

func CreateInventoryInfo(subprocessConfig *config.SubprocessConfig, collectVirtualInterfaces bool) []byte {
	d := util.NewDependencies(&subprocessConfig.DryRunConfig, "/host")

	ret := models.Inventory{
		BmcAddress:   GetBmcAddress(subprocessConfig, d),
		BmcV6address: GetBmcV6Address(subprocessConfig, d),
		Boot:         GetBoot(d),
		CPU:          GetCPU(d),
		Disks:        GetDisks(subprocessConfig, d),
		Gpus:         GetGPUs(d),
		Hostname:     GetHostname(d),
		Interfaces:   GetInterfaces(d, collectVirtualInterfaces),
		Memory:       GetMemory(d),
		SystemVendor: GetVendor(d),
		Routes:       GetRoutes(d),
		TpmVersion:   GetTPM(d),
	}

	if subprocessConfig.DryRunEnabled {
		applyDryRunConfig(subprocessConfig, &ret)
	}

	b, _ := json.Marshal(&ret)
	return b
}
