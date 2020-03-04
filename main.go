package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/filanov/bm-inventory/client"
	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/scanners"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	RETRY_SLEEP_SECS = 60
)

type Config struct {
	IsText bool
	TargetHost string
	TargetPort int
}

func printHelpAndExit() {
	fmt.Printf("Usage: %s [--help] [--text] [--host <host>] [--port <port>]\n", os.Args[0])
	os.Exit(0)
}

func processArgs() *Config {
	ret := &Config{}
	flag.BoolVar(&ret.IsText, "text", false, "Should text be displayed")
	flag.StringVar(&ret.TargetHost, "host", client.DefaultHost, "The target host")
	flag.IntVar(&ret.TargetPort, "port", 80, "The target port")
	h :=  flag.Bool("help", false, "Help message")
	flag.Parse()
	if h != nil && *h {
		printHelpAndExit()
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
			Serial: scanners.ReadMotherboadSerial(),
		},
	}
	return ret
}

func registerNodeWithRetry(cfg *Config) {
	bmInventory := createBmInventoryClient(cfg)
	for {
		_, err := bmInventory.Inventory.RegisterNode(context.Background(), createRegisterParams())
		if err == nil {
			return
		}
		log.Warnf("Error registering node: %s", err.Error())
		time.Sleep(RETRY_SLEEP_SECS * time.Second)
	}
}

func main() {
	cfg := processArgs()
	if cfg.IsText {
		fmt.Printf("%s", string(createNodeInfo()))
	} else {
		registerNodeWithRetry(cfg)
	}
}
