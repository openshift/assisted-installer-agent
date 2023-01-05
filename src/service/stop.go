package service

import (
	"fmt"

	"github.com/coreos/go-systemd/dbus"
)

// systemdUnitName is the name of the systemd unit of the agent.
const systemdUnitName = "agent.service"

// Stop stops the agent service using the equivalent of `systemctl stop agent.service`. If doesn't
// return when it succeeds, as the agent processes will be gone. Returns an error if something fails
// while trying to stop the service.
func Stop() error {
	// Open the D-Bus connection:
	dbusConn, err := dbus.NewSystemConnection()
	if err != nil {
		return fmt.Errorf("failed to get d-bus system connection: %v", err)
	}
	defer dbusConn.Close()

	// Send the request to stop the agent service:
	systemdResultCh := make(chan string)
	systemdJob, err := dbusConn.StopUnit(systemdUnitName, "replace", systemdResultCh)
	if err != nil {
		return fmt.Errorf(
			"failed to stop systemd unit '%s': %v",
			systemdUnitName, err,
		)
	}

	// Wait for the request to complete. Note that we trying to stop ourselves, so this will
	// most likely never run, we do it only to catch and report failures.
	systemdResult := <-systemdResultCh
	if systemdResult != "done" {
		return fmt.Errorf(
			"failed to stop systemd unit '%s', job is '%d' and result is '%s'",
			systemdUnitName, systemdJob, systemdResult,
		)
	}

	return nil
}
