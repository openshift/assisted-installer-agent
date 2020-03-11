package commands

import (
	"context"
	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/client"
	"github.com/ori-amizur/introspector/src/config"
	log "github.com/sirupsen/logrus"
	"time"
)

var stepType2Handler = map[models.StepType] func(string) (string, error) {
	models.StepTypeHardawareInfo: GetHardwareInfo,
	//models.StepTypeConnectivityCheck: connectivityCheckHandler,
}

func handleSingleStep(stepType models.StepType, data string, handler func(string) (string, error)) {
	output, err := handler(data)
	params := inventory.PostStepReplyParams{
		NodeID:     *CurrentNode.ID,
		Context:    nil,
		HTTPClient: nil,
	}
	reply := models.StepReply{
		Output:                output,
		StepType:              stepType,
		SucccessfulCompletion: err == nil,
	}
	if err != nil {
		reply.Error = err.Error()
	}
	params.Reply = &reply
	inventoryClient := client.CreateBmInventoryClient()
	_, err = inventoryClient.Inventory.PostStepReply(context.Background(), &params)
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
		go handleSingleStep(step.StepType, step.Data, handler)
	}
}

func ProcessSteps() {
	inventoryClient := client.CreateBmInventoryClient()
	for {
		params := inventory.GetNextStepsParams{
			NodeID: *CurrentNode.ID,
		}
		result, err := inventoryClient.Inventory.GetNextSteps(context.Background(), &params)
		if err != nil {
			log.Warnf("Could not query next steps: %s", err.Error())
			continue
		}
		handleSteps(result.Payload)
		time.Sleep(time.Duration(config.GlobalConfig.IntervalSecs) * time.Second)
	}
}
