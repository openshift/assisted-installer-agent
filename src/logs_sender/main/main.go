package main

import (
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/logs_sender"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	config.ProcessLogsSenderConfigArgs(true, true)
	util.SetLogging("logs-sender", config.LogsSenderConfig.TextLogging, config.LogsSenderConfig.JournalLogging)
	err, report := logs_sender.SendLogs(&logs_sender.LogsSenderExecuter{})
	if err != nil {
		fmt.Println("Failed to run send logs ", err.Error())
		os.Exit(-1)
	}
	fmt.Println("Logs were sent")
	fmt.Println(report)
}
