package main

import (
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/logs_sender"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func main() {
	loggingConfig := config.ProcessLogsSenderConfigArgs(true, true)
	config.ProcessDryRunArgs(&loggingConfig.DryRunConfig)
	util.SetLoggingWithStdOut("logs-sender", loggingConfig.TextLogging, loggingConfig.JournalLogging, loggingConfig.ForcedHostID)
	err, report := logs_sender.SendLogs(loggingConfig, logs_sender.NewLogsSenderExecuter(loggingConfig, loggingConfig.TargetURL,
		loggingConfig.PullSecretToken,
		loggingConfig.AgentVersion))
	if err != nil {
		fmt.Println("Failed to run send logs ", err.Error())
		os.Exit(-1)
	}
	fmt.Println("Logs were sent")
	fmt.Println(report)
}
