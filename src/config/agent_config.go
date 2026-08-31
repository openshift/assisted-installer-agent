package config

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

type AgentConfig struct {
	DryRunConfig
	ConnectivityConfig
	IntervalSecs         int
	HostID               string
	IronicStatusFilePath string
	LoggingConfig
}

func printHelpAndExit() {
	flag.CommandLine.Usage()
	os.Exit(0)
}

func ProcessArgs() *AgentConfig {
	ret := &AgentConfig{}

	err := RegisterDryRunArgs(&ret.DryRunConfig)
	if err != nil {
		log.Fatalf("Failed to register dry run arguments: %v", err)
	}

	RegisterLoggingArgs(&ret.LoggingConfig)

	flag.StringVar(&ret.TargetURL, "url", "", "The target URL, including a scheme and optionally a port (overrides the host and port arguments")
	flag.StringVar(&ret.InfraEnvID, "infra-env-id", "", "The value of infra-env-id")
	flag.StringVar(&ret.AgentVersion, "agent-version", "", "Full image reference of the agent, for example 'quay.io/edge-infrastructure/assisted-installer-agent:v2.5.2'")
	flag.IntVar(&ret.IntervalSecs, "interval", 60, "Interval between steps polling in seconds")
	flag.StringVar(&ret.CACertificatePath, "cacert", "", "Path to custom CA certificate in PEM format")
	flag.BoolVar(&ret.InsecureConnection, "insecure", false, "Do not validate TLS certificate")
	flag.StringVar(&ret.HostID, "host-id", "", "Host identification")
	flag.StringVar(&ret.IronicStatusFilePath, "ironic-status-file", "", "Path to ironic agent status file for readiness check")
	h := flag.Bool("help", false, "Help message")
	flag.Parse()

	// Set ironic status file path from CLI flag or environment variable
	if ret.IronicStatusFilePath == "" {
		ret.IronicStatusFilePath = os.Getenv("IRONIC_STATUS_FILE")
	}
	// If neither flag nor env var is set, use the default path
	if ret.IronicStatusFilePath == "" {
		ret.IronicStatusFilePath = "/run/ironic/status.json"
	}
	if h != nil && *h {
		printHelpAndExit()
	}

	if ret.TargetURL == "" {
		log.Fatalf("Must provide a target URL")
	}

	if ret.InfraEnvID == "" {
		log.Fatal("infra-env-id must be provided")
	}

	ret.PullSecretToken = os.Getenv("PULL_SECRET_TOKEN")
	if ret.PullSecretToken == "" {
		log.Warnf("Agent Authentication Token not set")
	}

	return ret
}
