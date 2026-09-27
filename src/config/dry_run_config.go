package config

import (
	"flag"
	"fmt"

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

// RegisterDryRunArgs must not be called more than once per process.
// Subsequent calls will panic.
func RegisterDryRunArgs(dryRunConfig *DryRunConfig) error {
	defaultValues := DryRunConfig{}

	err := envconfig.Process("dryconfig", &defaultValues)
	if err != nil {
		return fmt.Errorf("envconfig error: %v", err)
	}

	flag.BoolVar(&dryRunConfig.DryRunEnabled, "dry-run", defaultValues.DryRunEnabled, "Dry run avoids/fakes certain actions while communicating with the service")
	flag.StringVar(&dryRunConfig.ForcedHostID, "force-id", defaultValues.ForcedHostID, "The fake host ID to give to the host")
	flag.StringVar(&dryRunConfig.ForcedMacAddress, "force-mac", defaultValues.ForcedMacAddress, "The fake mac address to give to the first network interface")
	flag.StringVar(&dryRunConfig.ForcedHostname, "force-hostname", defaultValues.ForcedHostname, "The fake hostname to give to this host")
	flag.StringVar(&dryRunConfig.ForcedHostIPv4, "forced-ipv4", defaultValues.ForcedHostIPv4, "The fake ip address to give to the host's network interface")
	flag.StringVar(&dryRunConfig.FakeRebootMarkerPath, "fake-reboot-marker-path", defaultValues.FakeRebootMarkerPath, "A path whose existence indicates a fake reboot happened")

	return nil
}
