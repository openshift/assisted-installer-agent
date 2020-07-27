package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ori-amizur/introspector/src/commands"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	config.ProcessSubprocessArgs(false, true)
	util.SetLogging("connectivity-check", config.SubprocessConfig.TextLogging, config.SubprocessConfig.JournalLogging)
	if flag.NArg() != 1 {
		log.Warnf("Expecting exactly single argument to connectivity check. Recieved %d", len(os.Args)-1)
		os.Exit(-1)
	}
	stdout, stderr, exitCode := commands.ConnectivityCheck("", flag.Arg(0))
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
