package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// StatusFileContent represents the structure of the ironic agent status file
type StatusFileContent struct {
	Status string `json:"status"`
}

const (
	StatusInitializing    = "initializing"
	StatusWaitingForAgent = "waiting for assisted agent"
	StatusFinalizing      = "finalizing"
)

var (
	waitingLoop = 2 * time.Second
)

// ValidateStatusFile checks the ironic agent status file and waits for readiness if needed.
// It implements the following state machine:
// - File not found: Feature not implemented (skip wait)
// - Empty file: Feature available but initializing (wait indefinitely)
// - {"status": "initializing"}: Ironic agent starting up (wait indefinitely)
// - {"status": ""}: Transitional state (wait indefinitely)
// - {"status": "waiting for assisted agent"}: Safe to proceed (proceed immediately)
// - {"status": "finalizing"}: Ironic agent already finished (error - exit)
// - Any other content: Unexpected state (error - exit)
func ValidateStatusFile(filePath string) error {
	logger := logrus.StandardLogger()

	// Should not happen
	if filePath == "" {
		logger.Info("No ironic status file path configured, skipping readiness check")

		return nil
	}

	// Check if file exists
	_, err := os.Stat(filePath)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			logger.Info("Ironic status file not found, feature not implemented - proceeding immediately")

			return nil

		default:
			return fmt.Errorf("failed to check ironic status file (%s): %w", filePath, err)
		}
	}

	// File exists, read and validate its content
	logger.Info("Ironic status file found, checking readiness status")

	err = waitForIronicReadiness(filePath)
	if err != nil {
		return fmt.Errorf("failed to check ironic readiness: %w", err)
	}

	return nil
}

func waitForIronicReadiness(filePath string) error {
	logger := logrus.StandardLogger()

	for {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Handle empty file - feature is available but still initializing
		if len(content) == 0 {
			logger.Debug("Ironic status file is empty, ironic agent still initializing - waiting")
			time.Sleep(waitingLoop)

			continue
		}

		// Parse the JSON content
		statusContent := StatusFileContent{}

		err = json.Unmarshal(content, &statusContent)
		if err != nil {
			return fmt.Errorf("failed to unmarshal json: %w", err)
		}

		// Handle different status states
		switch statusContent.Status {
		case StatusInitializing:
			logger.Debug("Ironic agent is initializing - waiting for readiness")
			time.Sleep(waitingLoop)

			continue

		case StatusWaitingForAgent:
			logger.Info("Ironic agent is ready, assisted installer agent can proceed")

			return nil

		case StatusFinalizing:
			return errors.New("ironic agent is already finalizing")

		case "":
			logger.Debug("Ironic agent defined an empty state, waiting for readiness")
			time.Sleep(waitingLoop)

			continue

		default:
			return fmt.Errorf("unexpected ironic status: %s", statusContent.Status)
		}
	}
}
