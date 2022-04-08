package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
)

// DryRunConfig defines configuration of the agent's dry-run mode
type DryRunConfig struct {
	DryRunEnabled        bool   `envconfig:"DRY_ENABLE"`
	ForcedHostID         string `envconfig:"DRY_HOST_ID"`
	ForcedHostIPv4       string `envconfig:"DRY_FORCED_HOST_IPV4"`
	ForcedMacAddress     string `envconfig:"DRY_FORCED_MAC_ADDRESS"`
	ForcedHostname       string `envconfig:"DRY_FORCED_HOSTNAME"`
	FakeRebootMarkerPath string `envconfig:"DRY_FAKE_REBOOT_MARKER_PATH"`
}

var DefaultDryRunConfig = DryRunConfig{
	DryRunEnabled:        false,
	ForcedHostID:         "",
	ForcedHostIPv4:       "",
	ForcedMacAddress:     "",
	ForcedHostname:       "",
	FakeRebootMarkerPath: "",
}

func ProcessDryRunArgs(dryRunConfig *DryRunConfig) {
	err := envconfig.Process("dryconfig", &DefaultDryRunConfig)
	if err != nil {
		fmt.Printf("envconfig error: %v", err)
		os.Exit(1)
	}

	flag.BoolVar(&dryRunConfig.DryRunEnabled, "dry-run", DefaultDryRunConfig.DryRunEnabled, "Dry run avoids/fakes certain actions while communicating with the service")
	flag.StringVar(&dryRunConfig.ForcedHostID, "force-id", DefaultDryRunConfig.ForcedHostID, "The fake host ID to give to the host")
	flag.StringVar(&dryRunConfig.ForcedMacAddress, "force-mac", DefaultDryRunConfig.ForcedMacAddress, "The fake mac address to give to the first network interface")
	flag.StringVar(&dryRunConfig.ForcedHostname, "force-hostname", DefaultDryRunConfig.ForcedHostname, "The fake hostname to give to this host")
	flag.StringVar(&dryRunConfig.ForcedHostIPv4, "forced-ipv4", DefaultDryRunConfig.ForcedHostIPv4, "The fake ip address to give to the host's network interface")
	flag.StringVar(&dryRunConfig.FakeRebootMarkerPath, "fake-reboot-marker-path", DefaultDryRunConfig.FakeRebootMarkerPath, "A path whose existence indicates a fake reboot happened")
	flag.Parse()
}
