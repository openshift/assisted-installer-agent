package commands

import (
	"context"
	"fmt"
	"github.com/ori-amizur/introspector/src/util"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/filanov/bm-inventory/client/inventory"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/client"
	"github.com/ori-amizur/introspector/src/config"
)

type HandlerType func(string, []string) (string, string, int)

var stepType2Handler = map[models.StepType]HandlerType{
	models.StepTypeHardwareInfo:      GetHardwareInfo,
	models.StepTypeConnectivityCheck: ConnectivityCheck,
	models.StepTypeExecute:           Execute,
}

func sendStepReply(ctx context.Context, stepID, output, errStr string, exitCode int) {
	logger := util.ToLogger(ctx)
	logger.Infof("Sending step <%s> reply output <%s> error <%s> exit-code <%d>", stepID, output, errStr, exitCode)
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
		logger.Warnf("Error posting step reply: %s", err.Error())
	}
}

func handleSingleStep(ctx context.Context, stepID string, command string, args []string, handler HandlerType) {
	output, errStr, exitCode := handler(command, args)
	sendStepReply(ctx, stepID, output, errStr, exitCode)
}

func handleSteps(ctx context.Context, steps models.Steps) {
	for _, step := range steps {
		handler, ok := stepType2Handler[step.StepType]
		if !ok {
			errStr := fmt.Sprintf("Unexpected step type: %s", step.StepType)
			util.ToLogger(ctx).Warn(errStr)
			sendStepReply(ctx, step.StepID, "", errStr, -1)
			continue
		}
		go handleSingleStep(ctx, step.StepID, step.Command, step.Args, handler)
	}
}

func ProcessSteps() {
	inventoryClient := client.CreateBmInventoryClient()
	for {
		params := inventory.GetNextStepsParams{
			HostID:    *CurrentHost.ID,
			ClusterID: strfmt.UUID(config.GlobalConfig.ClusterID),
		}
		ctx := client.NewContext()
		logger := util.ToLogger(ctx)
		logger.Info("Query for next steps")
		result, err := inventoryClient.Inventory.GetNextSteps(ctx, &params)
		if err != nil {
			logger.Warnf("Could not query next steps: %s", err.Error())
		} else {
			handleSteps(ctx, result.Payload)
		}
		time.Sleep(time.Duration(config.GlobalConfig.IntervalSecs) * time.Second)
	}
}
