package config

import (
	"flag"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

const agentVersionTagDelimiter = ":"

var GlobalAgentConfig struct {
	ConnectivityConfig
	IntervalSecs int
	HostID       string
	V2           bool
	LoggingConfig
}

func printHelpAndExit() {
	flag.CommandLine.Usage()
	os.Exit(0)
}

func ProcessArgs() {
	ret := &GlobalAgentConfig
	flag.StringVar(&ret.TargetURL, "url", "", "The target URL, including a scheme and optionally a port (overrides the host and port arguments")
	flag.StringVar(&ret.ClusterID, "cluster-id", "", "The value of the cluster-id")
	flag.StringVar(&ret.InfraEnvID, "infra-env-id", "", "The value of infra-env-id")
	flag.StringVar(&ret.AgentVersion, "agent-version", "", "Discovery agent version")
	flag.IntVar(&ret.IntervalSecs, "interval", 60, "Interval between steps polling in seconds")
	flag.BoolVar(&ret.JournalLogging, "with-journal-logging", true, "Use journal logging")
	flag.BoolVar(&ret.TextLogging, "with-text-logging", false, "Output log to file")
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

	if ret.ClusterID == "" && ret.InfraEnvID == "" {
		log.Fatal("One of cluster-id, infra-env-id must be provided")
	}

	if ret.ClusterID != "" && ret.InfraEnvID != "" {
		log.Fatal("Only one of cluster-id, infra-env-id must be provided")
	}

	ret.V2 = ret.InfraEnvID != ""

	ret.PullSecretToken = os.Getenv("PULL_SECRET_TOKEN")
	if ret.PullSecretToken == "" {
		log.Warnf("Agent Authentication Token not set")
	}

	// When given <image_url>:<tag> format, AgentVersion should point to the image tag.
	// In a case of multiple delimiters, grab the rightmost slice.
	// Otherwise, we leave the agent-version str intact.
	agentVersionTag := strings.Split(ret.AgentVersion, agentVersionTagDelimiter)
	ret.AgentVersion = agentVersionTag[len(agentVersionTag)-1]
}
