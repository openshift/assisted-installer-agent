package commands

import (
	"context"
	"time"

	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/client"
	"github.com/ori-amizur/introspector/src/config"
	log "github.com/sirupsen/logrus"
)

type HandlerType func(string, []string) (string, string, int)

var stepType2Handler = map[models.StepType]HandlerType{
	models.StepTypeHardawareInfo:     GetHardwareInfo,
	models.StepTypeConnectivityCheck: ConnectivityCheck,
	models.StepTypeExecute: 		  Execute,
}

func handleSingleStep(stepID string, command string, args []string, handler HandlerType) {
	output, errStr, exitCode := handler(command, args)
	params := inventory.PostStepReplyParams{
		HostID:     *CurrentHost.ID,
		Context:    nil,
		HTTPClient: nil,
	}
	reply := models.StepReply{
		Output:   output,
		StepID:stepID,
		ExitCode:int64(exitCode),
		Error:errStr,
	}
	params.Reply = &reply
	inventoryClient := client.CreateBmInventoryClient()
	_, err := inventoryClient.Inventory.PostStepReply(context.Background(), &params)
	if err != nil {
		log.Warnf("Error posting step reply: %s")
	}
}

func handleSteps(steps models.Steps) {
	for _, step := range steps {
		handler, ok := stepType2Handler[step.StepType]
		if !ok {
			log.Warnf("Unexpected step type: %s", step.StepType)
			continue
		}
		go handleSingleStep(step.StepID,  step.Command, step.Args, handler)
	}
}

func ProcessSteps() {
	inventoryClient := client.CreateBmInventoryClient()
	for {
		params := inventory.GetNextStepsParams{
			HostID: *CurrentHost.ID,
		}
		result, err := inventoryClient.Inventory.GetNextSteps(context.Background(), &params)
		if err != nil {
			log.Warnf("Could not query next steps: %s", err.Error())
		} else {
			handleSteps(result.Payload)
		}
		time.Sleep(time.Duration(config.GlobalConfig.IntervalSecs) * time.Second)
	}
}
