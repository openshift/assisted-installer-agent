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

func processRequestArg() string {
	var executableConfig Config
	ret := &executableConfig
	flag.StringVar(&ret.Request, "request", "",
		"The request details. See models.DomainResolutionRequest")
	flag.Parse()

	if executableConfig.Request == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}
	return executableConfig.Request
}

func main() {
	request := processRequestArg()
	subprocessConfig := config.ProcessSubprocessArgs(config.DefaultLoggingConfig)
	config.ProcessDryRunArgs(&subprocessConfig.DryRunConfig)

	util.SetLogging("domain_resolution",
		subprocessConfig.TextLogging,
		subprocessConfig.JournalLogging, subprocessConfig.StdoutLogging, subprocessConfig.ForcedHostID)

	log.StandardLogger().Infof("Processing domain resolution, requested domains: %s", request)

	stdout, stderr, exitCode := domain_resolution.Run(request,
		&domain_resolution.DomainResolver{}, log.StandardLogger())

	_, _ = fmt.Fprint(os.Stdout, stdout)
	_, _ = fmt.Fprint(os.Stderr, stderr)

	os.Exit(exitCode)
}
