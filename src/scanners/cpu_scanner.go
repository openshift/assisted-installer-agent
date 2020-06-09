package scanners

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/filanov/bm-inventory/models"
	log "github.com/sirupsen/logrus"
)

func ReadCpus() *models.CPUDetails {
	cmd := exec.Command("lscpu")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running lscpu: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	r := regexp.MustCompile("^([^:]+):[ \t]+([^ \t].*)$")
	ret := &models.CPUDetails{}
	for _, line := range lines {
		matches := r.FindStringSubmatch(line)
		if len(matches) == 3 {
			switch matches[1] {
			case "Architecture":
				ret.Architecture = matches[2]
			case "Model name":
				ret.ModelName = matches[2]
			case "CPU(s)":
				ret.Cpus, _ = strconv.ParseInt(matches[2], 10, 64)
			case "Thread(s) per core":
				ret.ThreadsPerCore, _ = strconv.ParseInt(matches[2], 10, 64)
			case "Socket(s)":
				ret.Sockets, _ = strconv.ParseInt(matches[2], 10, 64)
			case "CPU MHz":
				ret.CPUMhz, _ = strconv.ParseFloat(matches[2], 64)
			}
		}
	}
	return ret
}
