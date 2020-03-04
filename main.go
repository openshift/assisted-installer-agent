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
	"strconv"
)

type Config struct {
	IsText bool
	TargetHost string
	TargetPort int
}

func processArgs() *Config {
	ret := &Config{
		IsText:     false,
		TargetHost: client.DefaultHost,
		TargetPort: 80,
	}
	args := os.Args[1:]
	for i := 0; i < len(args) ; {
		switch args[i] {
		case "--text":
			ret.IsText = true
			i++
		case "--host":
			if i < len(args) -1 {
				ret.TargetHost = args[i + 1]
			}
			i += 2
		case "--port":
			if i < len(args) -1 {
				port, err := strconv.Atoi(args[i + 1])
				if err != nil {
					log.Fatalf("Bad port argument %s", args[i + 1])
				}
				ret.TargetPort = port
			}
			i += 2
		default:
			log.Fatalf("Unknown arg %s", args[i])
		}
	}
	return ret
}

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

func createUrl(cfg *Config) string {
	return fmt.Sprintf("http://%s:%d/%s", cfg.TargetHost, cfg.TargetPort, client.DefaultBasePath)
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

func createBmInventoryClient(cfg *Config) *client.BMInventory {
	clientConfig := client.Config{}
	clientConfig.URL,_  = url.Parse(createUrl(cfg))
	clientConfig.Transport = &RequestRoundTripper{next: http.DefaultTransport}
	bmInventory := client.New(clientConfig)
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
	cfg := processArgs()
	if cfg.IsText {
		fmt.Printf("%s", string(createNodeInfo()))
	} else {
		bmInventory := createBmInventoryClient(cfg)
		_, err := bmInventory.Inventory.RegisterNode(context.Background(), createRegisterParams())
		if err != nil {
			log.Warnf("Could not register node: %s", err.Error())
		}
	}
}
