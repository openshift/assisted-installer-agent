package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/client"
	"github.com/ori-amizur/introspector/src/config"
	log "github.com/sirupsen/logrus"
)

type HandlerType func(string, []string) (string, string, int)

var stepType2Handler = map[models.StepType]HandlerType{
	models.StepTypeHardwareInfo:      GetHardwareInfo,
	models.StepTypeConnectivityCheck: ConnectivityCheck,
	models.StepTypeExecute:           Execute,
}

func sendStepReply(stepID, output, errStr string, exitCode int) {
	log.Infof("Sending step <%s> reply output <%s> error <%s> exit-code <%d>", stepID, output, errStr, exitCode)
	params := inventory.PostStepReplyParams{
		HostID:    *CurrentHost.ID,
		ClusterID: strfmt.UUID(config.GlobalConfig.ClusterID),
	}
	reply := models.StepReply{
		Output:   output,
		StepID:   stepID,
		ExitCode: int64(exitCode),
		Error:    errStr,
	}
	params.Reply = &reply
	inventoryClient := client.CreateBmInventoryClient()
	_, err := inventoryClient.Inventory.PostStepReply(context.Background(), &params)
	if err != nil {
		log.Warnf("Error posting step reply: %s", err.Error())
	}
}

func handleSingleStep(stepID string, command string, args []string, handler HandlerType) {
	output, errStr, exitCode := handler(command, args)
	sendStepReply(stepID, output, errStr, exitCode)
}

func handleSteps(steps models.Steps) {
	for _, step := range steps {
		handler, ok := stepType2Handler[step.StepType]
		if !ok {
			errStr := fmt.Sprintf("Unexpected step type: %s", step.StepType)
			log.Warn(errStr)
			sendStepReply(step.StepID, "", errStr, -1)
			continue
		}
		go handleSingleStep(step.StepID, step.Command, step.Args, handler)
	}
}

func ProcessSteps() {
	inventoryClient := client.CreateBmInventoryClient()
	for {
		params := inventory.GetNextStepsParams{
			HostID:    *CurrentHost.ID,
			ClusterID: strfmt.UUID(config.GlobalConfig.ClusterID),
		}
		log.Info("Query for next steps")
		result, err := inventoryClient.Inventory.GetNextSteps(context.Background(), &params)
		if err != nil {
			log.Warnf("Could not query next steps: %s", err.Error())
		} else {
			handleSteps(result.Payload)
		}
		time.Sleep(time.Duration(config.GlobalConfig.IntervalSecs) * time.Second)
	}
}
