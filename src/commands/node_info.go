package commands

import (
	"encoding/json"
	"github.com/ori-amizur/introspector/src/scanners"
)

type NodeInfo struct {
	Cpu *scanners.CpuInfo                   `json:"cpu"`
	BlockDevices []scanners.BlockDeviceInfo `json:"block_devices"`
	Memory []scanners.MemoryInfo            `json:"memory"`
	Nics []scanners.NicInfo                 `json:"nics"`
}


func CreateNodeInfo() [] byte {
	info := NodeInfo{
		Cpu:          scanners.ReadCpus(),
		BlockDevices: scanners.ReadBlockDevices(),
		Memory:       scanners.ReadMemory(),
		Nics:         scanners.ReadNics(),
	}
	b, _ := json.Marshal(&info)
	return b
}
