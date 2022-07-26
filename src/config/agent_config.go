package config

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

type AgentConfig struct {
	DryRunConfig
	ConnectivityConfig
	IntervalSecs int
	HostID       string
	LoggingConfig
}

func printHelpAndExit() {
	flag.CommandLine.Usage()
	os.Exit(0)
}

func ProcessArgs() *AgentConfig {
	ret := &AgentConfig{}
	flag.StringVar(&ret.TargetURL, "url", "", "The target URL, including a scheme and optionally a port (overrides the host and port arguments")
	flag.StringVar(&ret.InfraEnvID, "infra-env-id", "", "The value of infra-env-id")
	flag.StringVar(&ret.AgentVersion, "agent-version", "", "Full image reference of the agent, for example 'quay.io/edge-infrastructure/assisted-installer-agent:v2.5.2'")
	flag.IntVar(&ret.IntervalSecs, "interval", 60, "Interval between steps polling in seconds")
	flag.BoolVar(&ret.JournalLogging, "with-journal-logging", true, "Use journal logging")
	flag.BoolVar(&ret.TextLogging, "with-text-logging", false, "Output log to file")
	flag.BoolVar(&ret.StdoutLogging, "with-stdout-logging", false, "Output log to stdout")
	flag.StringVar(&ret.CACertificatePath, "cacert", "", "Path to custom CA certificate in PEM format")
	flag.BoolVar(&ret.InsecureConnection, "insecure", false, "Do not validate TLS certificate")
	flag.StringVar(&ret.HostID, "host-id", "", "Host identification")
	h := flag.Bool("help", false, "Help message")
	flag.Parse()
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
