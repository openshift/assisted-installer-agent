package config

import (
	"flag"
	"fmt"
	"os"
)

var LogsSenderConfig struct {
	TextLogging            bool
	JournalLogging         bool
	Tags                   []string
	Services               []string
	Since                  string
	HostID                 string
	ClusterID              string
	CleanWhenDone          bool
	TargetURL              string
	PullSecretToken        string
	IsBootstrap            bool
	InstallerGatherlogging bool
	MastersIPs             string
}

func ProcessLogsSenderConfigArgs(defaultTextLogging, defaultJournalLogging bool) {
	var leaveFiles bool
	flag.BoolVar(&LogsSenderConfig.JournalLogging, "with-journal-logging", defaultJournalLogging, "Use journal logging")
	flag.BoolVar(&LogsSenderConfig.TextLogging, "with-text-logging", defaultTextLogging, "Use text logging")
	flag.StringVar(&LogsSenderConfig.Since, "since", "5 hours ago", "Journalctl since flag, same format")
	flag.StringVar(&LogsSenderConfig.TargetURL, "url", "", "The target URL, including a scheme and optionally a port (overrides the host and port arguments")
	flag.StringVar(&LogsSenderConfig.ClusterID, "cluster-id", "", "The value of the cluster-id, required")
	flag.StringVar(&LogsSenderConfig.HostID, "host-id", "host-id", "The value of the host-id")
	flag.StringVar(&LogsSenderConfig.PullSecretToken, "pull-secret-token", "", "Pull secret token")
	flag.BoolVar(&leaveFiles, "dont-clean", false, "Don't delete all created files on finish. Required")
	flag.BoolVar(&LogsSenderConfig.IsBootstrap, "bootstrap", false, "Gather and send logs on bootstrap node")
	flag.BoolVar(&LogsSenderConfig.InstallerGatherlogging, "with-installer-gather-logging", false, "Use installer-gather logging")
	flag.StringVar(&GlobalAgentConfig.CACertificatePath, "cacert", "", "Path to custom CA certificate in PEM format")
	flag.BoolVar(&GlobalAgentConfig.InsecureConnection, "insecure", false, "Do not validate TLS certificate")
	flag.StringVar(&LogsSenderConfig.MastersIPs, "masters-ips", "", "list of ',' separated IPs of all masters nodes in the cluster for SSH use")
	h := flag.Bool("help", false, "Help message")

	flag.Parse()

	LogsSenderConfig.CleanWhenDone = !leaveFiles

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

	LogsSenderConfig.Tags = []string{"agent", "installer"}
	if LogsSenderConfig.IsBootstrap {
		LogsSenderConfig.Services = []string{"bootkube"}
	}

	if LogsSenderConfig.PullSecretToken == "" {
		LogsSenderConfig.PullSecretToken = os.Getenv("PULL_SECRET_TOKEN")
	}
}
