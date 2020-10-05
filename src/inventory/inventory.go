package inventory

import (
	"encoding/json"
	"github.com/openshift/assisted-service/models"
	"time"
)

func ReadInventory() *models.Inventory {
	d := newDepedencies()
	ret := models.Inventory{
		BmcAddress:   GetBmcAddress(d),
		BmcV6address: GetBmcV6Address(d),
		Boot:         GetBoot(d),
		CPU:          GetCPU(d),
		Disks:        GetDisks(d),
		Hostname:     GetHostname(d),
		Interfaces:   GetInterfaces(d),
		Memory:       GetMemory(d),
		SystemVendor: GetVendor(d),
		Timestamp:    time.Now().Unix(),
	}
	return &ret
}

func CreateInveroryInfo() []byte {
	in := ReadInventory()
	b, _ := json.Marshal(&in)
	return b
}
