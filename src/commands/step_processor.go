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

func (s *stepSession) sendStepReply(stepID, output, errStr string, exitCode int) {
	s.Logger().Infof("Sending step <%s> reply output <%s> error <%s> exit-code <%d>", stepID, output, errStr, exitCode)
	params := installer.PostStepReplyParams{
		HostID:    *CurrentHost.ID,
		ClusterID: strfmt.UUID(config.GlobalAgentConfig.ClusterID),
	}
	reply := models.StepReply{
		Output:   output,
		StepID:   stepID,
		ExitCode: int64(exitCode),
		Error:    errStr,
	}
	params.Reply = &reply
	_, err := s.Client().Installer.PostStepReply(s.Context(), &params)
	if err != nil {
		s.Logger().Warnf("Error posting step reply: %s", err.Error())
	}
}

func (s *stepSession) handleSingleStep(stepID string, command string, args []string, handler HandlerType) {
	stdout, stderr, exitCode := handler(command, args...)
	s.sendStepReply(stepID, stdout, stderr, exitCode)
}

func (s *stepSession) handleSteps(steps models.Steps) {
	for _, step := range steps {
		handler, ok := stepType2Handler[step.StepType]
		if !ok {
			errStr := fmt.Sprintf("Unexpected step type: %s", step.StepType)
			s.Logger().Warn(errStr)
			s.sendStepReply(step.StepID, "", errStr, -1)
			continue
		}
		go s.handleSingleStep(step.StepID, step.Command, step.Args, handler)
	}
}

func (s *stepSession) processSingleSession() {
	params := installer.GetNextStepsParams{
		HostID:    *CurrentHost.ID,
		ClusterID: strfmt.UUID(config.GlobalAgentConfig.ClusterID),
	}
	s.Logger().Info("Query for next steps")
	result, err := s.Client().Installer.GetNextSteps(s.Context(), &params)
	if err != nil {
		s.Logger().Warnf("Could not query next steps: %s", err.Error())
	} else {
		s.handleSteps(result.Payload)
	}

}

func ProcessSteps() {
	for {
		s := newSession()
		s.processSingleSession()
		time.Sleep(time.Duration(config.GlobalAgentConfig.IntervalSecs) * time.Second)
	}
}
