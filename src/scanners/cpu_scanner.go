package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type CpuInfo struct {
	Architecture string `json:"architecture"`
	ModelName string  `json:"model_name"`
	Cpus  int			`json:"cpus"`
	ThreadsPerCore int  `json:"threads_per_core"`
	Sockets int  `json:"sockets"`
	CPUMhz float64  `json:"cpu_mhz"`
}

func ReadCpus() *CpuInfo {
	cmd := exec.Command("lscpu")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running lscpu: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	r := regexp.MustCompile("^([^:]+):[ \t]+([^ \t].*)$")
	ret := &CpuInfo{}
	for _ , line := range lines {
		matches := r.FindStringSubmatch(line)
		if len(matches) == 3 {
			switch matches[1] {
			case "Architecture":
				ret.Architecture = matches[2]
			case "Model name":
				ret.ModelName = matches[2]
			case "CPU(s)":
				ret.Cpus, _ = strconv.Atoi(matches[2])
			case "Thread(s) per core":
				ret.ThreadsPerCore, _ = strconv.Atoi(matches[2])
			case "Socket(s)":
				ret.Sockets, _ = strconv.Atoi(matches[2])
			case "CPU MHz":
				ret.CPUMhz, _ = strconv.ParseFloat(matches[2], 64)
			}
		}
	}
	return ret
}