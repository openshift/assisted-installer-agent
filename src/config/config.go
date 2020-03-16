package config

import (
	"os"
	"flag"
	"github.com/filanov/bm-inventory/client"
)

type Config struct {
	IsText bool
	TargetHost string
	TargetPort int
	Namespace string
	IntervalSecs int
	ConnectivityParams string
}

var GlobalConfig Config

func printHelpAndExit() {
	flag.CommandLine.Usage()
	os.Exit(0)
}

func ProcessArgs()  {
	ret := &GlobalConfig
	flag.BoolVar(&ret.IsText, "text", false, "Output only as text")
	flag.StringVar(&ret.TargetHost, "host", client.DefaultHost, "The target host")
	flag.IntVar(&ret.TargetPort, "port", 80, "The target port")
	flag.StringVar(&ret.Namespace, "namespace", "default-namespace", "The value of the namespace")
	flag.IntVar(&ret.IntervalSecs, "interval", 60, "Interval between steps polling in seconds")
	flag.StringVar(&ret.ConnectivityParams, "connectivity", "", "Test connectivity as output string")
	h :=  flag.Bool("help", false, "Help message")
	flag.Parse()
	if h != nil && *h {
		printHelpAndExit()
	}
}

