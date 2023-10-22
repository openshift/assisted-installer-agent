package logs_sender

import (
	"fmt"
	"os"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func Main() {
	loggingConfig := config.ProcessLogsSenderConfigArgs(true, true)
	config.ProcessDryRunArgs(&loggingConfig.DryRunConfig)
	util.SetLogging("logs-sender", loggingConfig.TextLogging, loggingConfig.JournalLogging, loggingConfig.StdoutLogging, loggingConfig.ForcedHostID)
	err, report := SendLogs(loggingConfig, NewLogsSenderExecuter(loggingConfig, loggingConfig.TargetURL,
		loggingConfig.PullSecretToken,
		loggingConfig.AgentVersion))
	if err != nil {
		fmt.Println("Failed to run send logs ", err.Error())
		os.Exit(-1)
	}
	fmt.Println("Logs were sent")
	fmt.Println(report)
}
