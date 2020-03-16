package commands

import (
	"encoding/json"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/scanners"
)

func CreateNodeInfo() [] byte {
	info := models.Introspection{
		BlockDevices: scanners.ReadBlockDevices(),
		CPU:          scanners.ReadCpus(),
		Memory:       scanners.ReadMemory(),
		Nics:         scanners.ReadNics(),
	}
	b, _ := json.Marshal(&info)
	return b
}
