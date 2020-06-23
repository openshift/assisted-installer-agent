package commands

import (
	"fmt"
	"time"

	"github.com/ori-amizur/introspector/src/util"

	"github.com/go-openapi/strfmt"

	"github.com/filanov/bm-inventory/client/installer"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/session"
)

type HandlerType func(string, ...string) (stdout string, stderr string, exitCode int)

var stepType2Handler = map[models.StepType]HandlerType{
	models.StepTypeHardwareInfo:      GetHardwareInfo,
	models.StepTypeConnectivityCheck: ConnectivityCheck,
	models.StepTypeExecute:           util.Execute,
	models.StepTypeInventory:         GetInventory,
}

type stepSession struct {
	session.InventorySession
}

func newSession() *stepSession {
	ret := stepSession{*session.New()}
	return &ret
}

func (s *stepSession) sendStepReply(stepType models.StepType, stepID, output, errStr string, exitCode int) {
	s.Logger().Infof("Sending step <%s> reply output <%s> error <%s> exit-code <%d>", stepID, output, errStr, exitCode)
	params := installer.PostStepReplyParams{
		HostID:    *CurrentHost.ID,
		ClusterID: strfmt.UUID(config.GlobalAgentConfig.ClusterID),
	}
	reply := models.StepReply{
		StepType: stepType,
		StepID:   stepID,
		ExitCode: int64(exitCode),
		Output:   output,
		Error:    errStr,
	}
	params.Reply = &reply
	_, err := s.Client().Installer.PostStepReply(s.Context(), &params)
	if err != nil {
		s.Logger().Warnf("Error posting step reply: %s", err.Error())
	}
}

func (s *stepSession) handleSingleStep(stepType models.StepType, stepID string, command string, args []string, handler HandlerType) {
	stdout, stderr, exitCode := handler(command, args...)
	s.sendStepReply(stepType, stepID, stdout, stderr, exitCode)
}

func (s *stepSession) handleSteps(steps *models.Steps) {
	for _, step := range steps.Instructions {
		var handler HandlerType
		if step.Command != "" {
			handler = util.Execute
		} else {
			// This part will be deprecated and will be removed after bm-inventory will be adapted to execute only commands
			handler = stepType2Handler[step.StepType]
			if handler == nil {
				errStr := fmt.Sprintf("Unexpected step type: %s", step.StepType)
				s.Logger().Warn(errStr)
				s.sendStepReply(step.StepType, step.StepID, "", errStr, -1)
				continue
			}
		}
		go s.handleSingleStep(step.StepType, step.StepID, step.Command, step.Args, handler)
	}
}

func (s *stepSession) processSingleSession() int64 {
	params := installer.GetNextStepsParams{
		HostID:    *CurrentHost.ID,
		ClusterID: strfmt.UUID(config.GlobalAgentConfig.ClusterID),
	}
	s.Logger().Info("Query for next steps")
	result, err := s.Client().Installer.GetNextSteps(s.Context(), &params)
	if err != nil {
		s.Logger().Warnf("Could not query next steps: %s", err.Error())
		return int64(config.GlobalAgentConfig.IntervalSecs)
	} else {
		s.handleSteps(result.Payload)
	}
	return result.Payload.NextInstructionSeconds
}

func ProcessSteps() {
	var nextRunIn int64
	for {
		s := newSession()
		nextRunIn = s.processSingleSession()
		time.Sleep(time.Duration(nextRunIn) * time.Second)
	}
}
