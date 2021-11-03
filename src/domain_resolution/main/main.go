package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/domain_resolution"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Request string
}

var executableConfig Config

func processArgs() {
	ret := &executableConfig
	flag.StringVar(&ret.Request, "request", "",
		"The request details. See models.DomainResolutionRequest")
	flag.Parse()

	if executableConfig.Request == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}
}

func main() {
	processArgs()
	config.ProcessDryRunArgs()
	config.ProcessSubprocessArgs(config.DefaultLoggingConfig)

	util.SetLogging("domain_resolution",
		config.SubprocessConfig.TextLogging,
		config.SubprocessConfig.JournalLogging, config.GlobalDryRunConfig.ForcedHostID)

	log.StandardLogger().Infof("Processing domain resolution, requested domains: %s", executableConfig.Request)

	stdout, stderr, exitCode := domain_resolution.Run(executableConfig.Request,
		&domain_resolution.DomainResolver{}, log.StandardLogger())

	_, _ = fmt.Fprint(os.Stdout, stdout)
	_, _ = fmt.Fprint(os.Stderr, stderr)

	os.Exit(exitCode)
}
