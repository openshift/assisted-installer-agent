package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"

	"github.com/openshift/assisted-installer-agent/src/fio_perf_check"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.ProcessArgs()
	util.SetLogging("fio-perf-check", config.GlobalAgentConfig.TextLogging, config.GlobalAgentConfig.JournalLogging)

	var fioPerfCheckRequest models.FioPerfCheckRequest
    if err := json.Unmarshal([]byte(flag.Arg(flag.NArg()-1)), &fioPerfCheckRequest); err != nil {
        log.Warnf("Expecting a valid request in json format as the last argument")
        os.Exit(-1)
    }
	perfCheck := fio_perf_check.NewPerfCheck(fio_perf_check.NewDependencies())
	stdout, stderr, exitCode := perfCheck.FioPerfCheck(flag.Arg(len(flag.Args())-1), log.StandardLogger())
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
