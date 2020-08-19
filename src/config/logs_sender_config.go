package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type tagsList []string

func (t *tagsList) String() string {
	return strings.Join(*t, " ")
}

func (t *tagsList) ToStringList() []string {
	return *t
}

func (t *tagsList) Set(value string) error {
	*t = append(*t, value)
	return nil
}

var LogsSenderConfig struct {
	TextLogging     bool
	JournalLogging  bool
	Tags            []string
	Since           string
	HostID          string
	ClusterID       string
	CleanWhenDone   bool
	TargetURL       string
	PullSecretToken string
}

func ProcessLogsSenderConfigArgs(defaultTextLogging, defaultJournalLogging bool) {
	var tags tagsList
	var leaveFiles bool
	flag.BoolVar(&LogsSenderConfig.JournalLogging, "with-journal-logging", defaultJournalLogging, "Use journal logging")
	flag.BoolVar(&LogsSenderConfig.TextLogging, "with-text-logging", defaultTextLogging, "Use text logging")
	flag.Var(&tags, "tag", "Journalctl tag to filter")
	flag.StringVar(&LogsSenderConfig.Since, "since", "5 hours ago", "Journalctl since flag, same format")
	flag.StringVar(&LogsSenderConfig.TargetURL, "url", "", "The target URL, including a scheme and optionally a port (overrides the host and port arguments")
	flag.StringVar(&LogsSenderConfig.ClusterID, "cluster-id", "", "The value of the cluster-id, required")
	flag.StringVar(&LogsSenderConfig.HostID, "host-id", "host-id", "The value of the host-id")
	flag.StringVar(&LogsSenderConfig.PullSecretToken, "pull-secret-token", "", "Pull secret token")
	flag.BoolVar(&leaveFiles, "dont-clean", false, "Don't delete all created files on finish. Required")
	flag.StringVar(&GlobalAgentConfig.CACertificatePath, "cacert", "", "Path to custom CA certificate in PEM format")
	flag.BoolVar(&GlobalAgentConfig.InsecureConnection, "insecure", false, "Do not validate TLS certificate")
	h := flag.Bool("help", false, "Help message")

	flag.Parse()

	LogsSenderConfig.CleanWhenDone = !leaveFiles

	if h != nil && *h {
		printHelpAndExit()
	}

	required := []string{"host-id", "cluster-id", "url", "tag"}
	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			fmt.Fprintf(os.Stderr, "missing required -%s argument/flag\n", req)
			os.Exit(2) // the same exit code flag.Parse uses
		}
	}

	LogsSenderConfig.Tags = tags.ToStringList()

	if LogsSenderConfig.PullSecretToken == "" {
		LogsSenderConfig.PullSecretToken = os.Getenv("PULL_SECRET_TOKEN")
	}
	if LogsSenderConfig.PullSecretToken == "" {
		_, _ = fmt.Fprint(os.Stderr, "missing required -pull-secret-token argument or set PULL_SECRET_TOKEN os env\n")
		printHelpAndExit()
	}
}
