package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strings"
)

func readMachineId() *string {
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
	return nil

}

func readMotherboadSerial() *string {
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
	return nil
}

func ReadId() *string {
	ret := readMotherboadSerial()
	if ret == nil {
		ret = readMachineId()
	}
	if ret == nil {
		missing := "Missing id"
		ret = &missing
	}
	return ret
}


