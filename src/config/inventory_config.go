package config

import (
	"flag"

	log "github.com/sirupsen/logrus"
)

// Inventory command configuration
type InventoryConfig struct {
	DryRunConfig
	LoggingConfig
	GPUConfigFile string
}

func ProcessInventoryConfigArgs() *InventoryConfig {
	ret := &InventoryConfig{}

	RegisterLoggingArgs(&ret.LoggingConfig)

	err := RegisterDryRunArgs(&ret.DryRunConfig)
	if err != nil {
		log.Fatalf("Failed to register dry run arguments: %v", err)
	}

	flag.StringVar(&ret.GPUConfigFile, "gpu-config-file", "", "Configuration file for GPU discovery")
	h := flag.Bool("help", false, "Help message")
	flag.Parse()

	if h != nil && *h {
		printHelpAndExit()
	}

	return ret
}
