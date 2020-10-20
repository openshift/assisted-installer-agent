package config

import (
	"flag"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

const agentVersionTagDelimiter = ":"

var GlobalAgentConfig struct {
	IsText             bool
	TargetURL          string
	ClusterID          string
	IntervalSecs       int
	ConnectivityParams string
	InventoryImage     string
	JournalLogging     bool
	TextLogging        bool
	AgentVersion       string
	PullSecretToken    string
	InsecureConnection bool
	CACertificatePath  string
}

func printHelpAndExit() {
	flag.CommandLine.Usage()
	os.Exit(0)
}

func ProcessArgs() {
	ret := &GlobalAgentConfig
	flag.BoolVar(&ret.IsText, "text", false, "Output only as text")
	flag.StringVar(&ret.TargetURL, "url", "", "The target URL, including a scheme and optionally a port (overrides the host and port arguments")
	flag.StringVar(&ret.ClusterID, "cluster-id", "default-cluster", "The value of the cluster-id")
	flag.StringVar(&ret.AgentVersion, "agent-version", "", "Discovery agent version")
	flag.IntVar(&ret.IntervalSecs, "interval", 60, "Interval between steps polling in seconds")
	flag.StringVar(&ret.ConnectivityParams, "connectivity", "", "Test connectivity as output string")
	flag.StringVar(&ret.InventoryImage, "inventory-image", "quay.io/ocpmetal/inventory:latest", "The image of inventory")
	flag.BoolVar(&ret.JournalLogging, "with-journal-logging", true, "Use journal logging")
	flag.BoolVar(&ret.TextLogging, "with-text-logging", false, "Output log to file")
	flag.StringVar(&ret.CACertificatePath, "cacert", "", "Path to custom CA certificate in PEM format")
	flag.BoolVar(&ret.InsecureConnection, "insecure", false, "Do not validate TLS certificate")
	h := flag.Bool("help", false, "Help message")
	flag.Parse()
	if h != nil && *h {
		printHelpAndExit()
	}

	if ret.TargetURL == "" {
		log.Fatalf("Must provide a target URL")
	}

	ret.PullSecretToken = os.Getenv("PULL_SECRET_TOKEN")
	if ret.PullSecretToken == "" {
		log.Warnf("Missing Pull Secret Token environment variable")
	}

	// When given <image_url>:<tag> format, AgentVersion should point to the image tag.
	// In a case of multiple delimiters, grab the rightmost slice.
	// Otherwise, we leave the agent-version str intact.
	agentVersionTag := strings.Split(ret.AgentVersion, agentVersionTagDelimiter)
	ret.AgentVersion = agentVersionTag[len(agentVersionTag)-1]
}
