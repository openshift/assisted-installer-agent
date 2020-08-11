package util

import (
	bytes2 "bytes"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

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
	return string(stdoutBytes.Bytes()), getErrorStr(err, &stderrBytes), getExitCode(err)
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
