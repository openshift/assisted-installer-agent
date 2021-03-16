package inventory

import (
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/sirupsen/logrus"
)

func GetHostname(dependencies util.IDependencies) string {
	h, err := dependencies.Hostname()
	if err != nil {
		logrus.WithError(err).Warn("Could not retrieve hostname")
		return ""
	}
	return strings.TrimSpace(h)
}
