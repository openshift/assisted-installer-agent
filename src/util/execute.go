package util

import (
	bytes2 "bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
)

const TimeoutExitCode = 124

func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	switch value := err.(type) {
	case *exec.ExitError:
		return value.ExitCode()
	default:
		return -1
	}
}

func getErrorStr(err error, stderr *bytes2.Buffer) string {
	b := stderr.Bytes()
	if len(b) > 0 {
		return string(b)
	} else if err != nil {
		return err.Error()
	}
	return ""
}

func Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	cmd := exec.Command(command, args...)
	var stdoutBytes, stderrBytes bytes2.Buffer
	cmd.Stdout = &stdoutBytes
	cmd.Stderr = &stderrBytes
	err := cmd.Run()
	return stdoutBytes.String(), getErrorStr(err, &stderrBytes), getExitCode(err)
}

func LogPrivilegedCommandOutput(logfile *os.File, result error, commandDescription string, command string, args ...string) error {
	log.Infof(commandDescription)
	loglnToFile(logfile, commandDescription)

	if err := ExecutePrivilegedToFile(logfile, command, args...); err != nil {
		result = multierror.Append(result, err)
	}

	return result
}

func ExecutePrivilegedToFile(logfile *os.File, command string, args ...string) error {
	loglnToFile(logfile, fmt.Sprintf("%s %s", command, strings.Join(args[:], " ")))

	stdout, stderr, exitcode := ExecutePrivileged(command, args...)
	if stderr != "" || exitcode != 0 {
		loglnToFile(logfile, stderr)
		return fmt.Errorf("%s failed: %d %s\n", command, exitcode, stderr)
	}

	loglnToFile(logfile, stdout)
	return nil
}

func ExecuteOutputToFile(outputFilePath string, command string, args ...string) (stderr string, exitCode int) {
	cmd := exec.Command(command, args...)
	var stderrBytes bytes2.Buffer
	outfile, err := os.Create(outputFilePath)
	if err != nil {
		log.WithError(err).Errorf("Failed to create output file %s", outputFilePath)
		return err.Error(), -1
	}
	defer outfile.Close()

	cmd.Stdout = outfile
	cmd.Stderr = &stderrBytes
	err = cmd.Run()
	return getErrorStr(err, &stderrBytes), getExitCode(err)
}

func ExecuteShell(command string) (stdout string, stderr string, exitCode int) {
	return Execute("bash", "-c", command)
}

func ExecutePrivileged(command string, args ...string) (stdout string, stderr string, exitCode int) {
	// nsenter is used here to launch processes inside the container in a way that makes said processes feel
	// and behave as if they're running on the host directly rather than inside the container
	commandBase := "nsenter"

	arguments := []string{
		"--target", "1",
		// Entering the cgroup namespace is not required for podman on CoreOS (where the
		// agent typically runs), but it's needed on some Fedora versions and
		// some other systemd based systems. Those systems are used to run dry-mode
		// agents for load testing. If this flag is not used, Podman will sometimes
		// have trouble creating a systemd cgroup slice for new containers.
		"--cgroup",
		// The mount namespace is required for podman to access the host's container
		// storage
		"--mount",
		// TODO: Document why we need the IPC namespace
		"--ipc",
		// Network namespace is needed for accessing host networking information
		// during inventory collection
		"--net",
		"--",
		command,
	}

	arguments = append(arguments, args...)
	return Execute(commandBase, arguments...)
}

func loglnToFile(logfile *os.File, message string) {
	logToFile(logfile, fmt.Sprintf("%s\n", message))
}

func logToFile(logfile *os.File, message string) {
	_, err := logfile.WriteString(message)
	if err != nil {
		log.WithError(err).Errorf("Failed logging '%s' to log file", message)
	}
}
