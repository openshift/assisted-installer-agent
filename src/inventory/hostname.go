package inventory

import (
	"github.com/sirupsen/logrus"
	"strings"
)


func GetHostname(dependencies IDependencies) string {
	h, err := dependencies.Hostname()
	if err != nil {
		logrus.WithError(err).Warn("Could not retrieve hostname")
		return ""
	}
	return strings.TrimSpace(h)
}
