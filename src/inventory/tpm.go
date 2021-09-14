package inventory

import (
	"errors"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/sirupsen/logrus"
)

func GetTPM(dependencies util.IDependencies) string {

	stdOut, stdErr, exitCode := dependencies.Execute("cat", "/sys/class/tpm/tpm0/tpm_version_major")
	if exitCode != 0 {
		if strings.Contains(stdErr, "No such file or directory") {
			return "none"
		}
		logrus.WithError(errors.New(stdErr)).Warn("Error checking TPM version")
		return ""
	}

	switch stdOut {
	case "1":
		return "1.2"
	case "2":
		return "2.0"
	}

	return stdOut
}
