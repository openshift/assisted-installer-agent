package config

import (
	"flag"
	"fmt"
	"os"
)

type LogsSenderConfig struct {
	LoggingConfig
	AgentConfig
	Tags                   []string
	Services               []string
	Since                  string
	HostID                 string
	ClusterID              string
	InfraEnvID             string
	CleanWhenDone          bool
	TargetURL              string
	PullSecretToken        string
	IsBootstrap            bool
	InstallerGatherlogging bool
	MastersIPs             string
}

func ProcessLogsSenderConfigArgs(defaultTextLogging, defaultJournalLogging bool) *LogsSenderConfig {
	var leaveFiles bool
	loggingConfig := &LogsSenderConfig{}
	flag.BoolVar(&loggingConfig.JournalLogging, "with-journal-logging", defaultJournalLogging, "Use journal logging")
	flag.BoolVar(&loggingConfig.TextLogging, "with-text-logging", defaultTextLogging, "Use text logging")
	flag.StringVar(&loggingConfig.Since, "since", "", "Journalctl since flag, same format")
	flag.StringVar(&loggingConfig.TargetURL, "url", "", "The target URL, including a scheme and optionally a port (overrides the host and port arguments")
	flag.StringVar(&loggingConfig.ClusterID, "cluster-id", "", "The value of the cluster-id, required")
	flag.StringVar(&loggingConfig.InfraEnvID, "infra-env-id", "", "The value of the infra-env-id")
	flag.StringVar(&loggingConfig.HostID, "host-id", "host-id", "The value of the host-id")
	flag.StringVar(&loggingConfig.PullSecretToken, "pull-secret-token", "", "Pull secret token")
	flag.BoolVar(&leaveFiles, "dont-clean", false, "Don't delete all created files on finish. Required")
	flag.BoolVar(&loggingConfig.IsBootstrap, "bootstrap", false, "Gather and send logs on bootstrap node")
	flag.BoolVar(&loggingConfig.InstallerGatherlogging, "with-installer-gather-logging", false, "Use installer-gather logging")
	flag.StringVar(&loggingConfig.CACertificatePath, "cacert", "", "Path to custom CA certificate in PEM format")
	flag.BoolVar(&loggingConfig.InsecureConnection, "insecure", false, "Do not validate TLS certificate")
	flag.StringVar(&loggingConfig.MastersIPs, "masters-ips", "", "list of ',' separated IPs of all masters nodes in the cluster for SSH use")
	h := flag.Bool("help", false, "Help message")

	flag.Parse()

	loggingConfig.CleanWhenDone = !leaveFiles

	if h != nil && *h {
		printHelpAndExit()
	}

	required := []string{"host-id", "cluster-id", "url"}
	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			fmt.Fprintf(os.Stderr, "missing required -%s argument/flag\n", req)
			os.Exit(2) // the same exit code flag.Parse uses
		}
	}

	loggingConfig.Tags = []string{"agent", "installer"}
	loggingConfig.Services = []string{"ironic-agent"}
	if loggingConfig.IsBootstrap {
		loggingConfig.Services = append(loggingConfig.Services, "bootkube")
	}

	if loggingConfig.PullSecretToken == "" {
		loggingConfig.PullSecretToken = os.Getenv("PULL_SECRET_TOKEN")
	}
	return loggingConfig
}
