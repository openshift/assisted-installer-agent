package util

import (
	bytes2 "bytes"
	"os/exec"
)

func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	switch  value := err.(type) {
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

func ExecuteShell(command string) (stdout string, stderr string, exitCode int) {
	return Execute("bash", "-c", command)
}
