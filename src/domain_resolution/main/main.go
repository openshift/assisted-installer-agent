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

var request string

func main() {
	flag.StringVar(&request, "request", "", "The request details. See models.DomainResolutionRequest")
	subprocessConfig := config.ProcessSubprocessArgs()

	if request == "" {
		flag.CommandLine.Usage()
		os.Exit(1)
	}

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
