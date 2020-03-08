package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strings"
)

func ReadMotherboadSerial() *string {
	cmd := exec.Command("dmidecode", "-t", "2")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running dmidecode: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	r := regexp.MustCompile("^[ \t]+Serial Number: +([^ \t]+)$")
	for _, line := range lines {
		matches := r.FindStringSubmatch(line)
		if len (matches) == 2 {
			return &matches[1]
		}
	}
	log.Warn("Could not find motherboard serial number")
	ret := "Missing serial"
	return &ret
}
