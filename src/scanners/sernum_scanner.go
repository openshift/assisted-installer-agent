package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strings"
)

func ReadMotherboadSerial() *string {
	cmd := exec.Command("cat", "/etc/machine-id")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error searching for machine-id: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	r := regexp.MustCompile("^([a-f0-9]+)$")
	for _, line := range lines {
		matches := r.FindStringSubmatch(line)
		if len (matches) == 2 {
			return &matches[1]
		}
	}
	log.Warn("Could not find machine-id")
	ret := "Missing serial"
	return &ret
}
