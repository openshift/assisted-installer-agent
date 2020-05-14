package main

import (
	"fmt"
	"github.com/ori-amizur/introspector/src/commands"
	"github.com/ori-amizur/introspector/src/util"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	util.SetLogging("connectivity_check")
	if len(os.Args) != 2 {
		log.Warnf( "Expecting exactly single argument to connectivity check. Recieved %d", len(os.Args) - 1)
		os.Exit(-1)
	}
	stdout, stderr, exitCode := commands.ConnectivityCheck("", os.Args[1:]...)
	fmt.Fprint(os.Stdout, stdout)
	fmt.Fprint(os.Stderr, stderr)
	os.Exit(exitCode)
}
