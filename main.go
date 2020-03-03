package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filanov/bm-inventory/client"
	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/scanners"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
)

type NodeInfo struct {
	Cpu *scanners.CpuInfo `json:"cpu"`
	BlockDevices []scanners.BlockDeviceInfo `json:"block_devices"`
	Memory []scanners.MemoryInfo `json:"memory"`
	Interfaces []scanners.InterfaceInfo `json:"interfaces"`
	Addresses []scanners.AddressInfo `json:"addresses"`
}

type RequestRoundTripper struct {next http.RoundTripper}

func (rt *RequestRoundTripper)RoundTrip(req *http.Request) (*http.Response, error) {
	return rt.next.RoundTrip(req)
}

func createUrl() string {
	return "http://" + client.DefaultHost + "/" + client.DefaultBasePath
}

func createNodeInfo() [] byte {
	info := NodeInfo{
		Cpu:          scanners.ReadCpus(),
		BlockDevices: scanners.ReadBlockDevices(),
		Memory:       scanners.ReadMemory(),
		Interfaces:   scanners.ReadInterfaces(),
		Addresses:    scanners.ReadAddresses(),
	}
	b, _ := json.Marshal(&info)
	return b
}

func createBmInventoryClient() *client.BMInventory {
	cfg := client.Config{}
	cfg.URL,_  = url.Parse(createUrl())
	cfg.Transport = &RequestRoundTripper{next:http.DefaultTransport}
	bmInventory := client.New(cfg)
	return bmInventory
}

func createRegisterParams() *inventory.RegisterNodeParams {
	nodeInfo := string(createNodeInfo())
	namespace := "namespace"
	ret := &inventory.RegisterNodeParams{
		NewNodeParams: &models.NodeCreateParams{
			HardwareInfo: &nodeInfo,
			Namespace:    &namespace,
		},
	}
	return ret
}

func main() {
	args := os.Args[1:]
	if len(args) == 1 && args[0] == "--text" {
		fmt.Printf("%s", string(createNodeInfo()))
	} else {
		bmInventory := createBmInventoryClient()
		_, err := bmInventory.Inventory.RegisterNode(context.Background(), createRegisterParams())
		if err != nil {
			log.Warnf("Could not register node: %s", err.Error())
		}
	}
}
