package upgrade_agent

import (
	"encoding/json"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"golang.org/x/sync/semaphore"

	"github.com/sirupsen/logrus"
)

// Dependencies represents the the dependencies of the upgrade agent command. It is intended to be
// used in unit tests, where the implementation will be a mock.
//
//go:generate mockery --name Dependencies --inpackage
type Dependencies interface {
	ExecutePrivileged(command string, args ...string) (stdout, stderr string, exitCode int)
}

// RealDependencies contains the real implementations of the dependencies.
type RealDependencies struct {
}

func (d *RealDependencies) ExecutePrivileged(command string, args ...string) (stdout,
	stderr string, exitcode int) {
	return util.ExecutePrivileged(command, args...)
}

// pullSem is used to prevent multiple simultaneous executions of the command that downloads
// the image.
var pullSem = semaphore.NewWeighted(1)

func Run(requestStr string, dependencies Dependencies, log logrus.FieldLogger) (stdout,
	stderr string, exitCode int) {
	// Deserialize the request:
	var request models.UpgradeAgentRequest
	err := json.Unmarshal([]byte(requestStr), &request)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"request": requestStr,
		}).Error("Failed to upgrade agent request string")
		return "", err.Error(), -1
	}

	// Create a logger containing a field for the image name, so that we don't have to repeat
	// that in all the log messages below:
	log = log.WithFields(logrus.Fields{
		"image": request.AgentImage,
	})

	// Prepare the response and rember to serialize and return it regardless of what happens
	// later within this function:
	response := models.UpgradeAgentResponse{
		AgentImage: request.AgentImage,
	}
	defer func() {
		responseBytes, marshalErr := json.Marshal(response)
		if marshalErr != nil {
			log.WithError(marshalErr).WithFields(logrus.Fields{
				"result": response.Result,
			}).Error("Failed to marshal response")
			stdout, stderr, exitCode = "", marshalErr.Error(), -1
		}
		stdout, stderr, exitCode = string(responseBytes), "", 0
	}()

	// If the semaphore is already acquired then return inmediately, as that means that
	// another image pull is already in progress:
	pullAllowed := pullSem.TryAcquire(1)
	if !pullAllowed {
		log.Info("Image pull is in progress")
		return
	}
	defer pullSem.Release(1)

	// Pull the image:
	log.Info("Pulling image")
	pullStdout, pullStderr, pullExitCode := dependencies.ExecutePrivileged(
		"podman", "pull", request.AgentImage,
	)
	log = log.WithFields(logrus.Fields{
		"stdout": pullStdout,
		"stderr": pullStderr,
		"code":   pullExitCode,
	})
	if pullExitCode == 0 {
		log.Info("Successfully pulled image")
		response.Result = models.UpgradeAgentResultSuccess
	} else {
		log.Error("Failed to pull image")
		response.Result = models.UpgradeAgentResultFailure
	}

	return
}
