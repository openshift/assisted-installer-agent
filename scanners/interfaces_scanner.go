package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type InterfaceInfo struct {
	Name string `json:"name"`
	State string `json:"state"`
	Mtu int `json:"mtu"`
	Mac string `json:"mac"`
}

func ReadInterfaces() [] InterfaceInfo {
	cmd := exec.Command("ip", "-o",  "link", "list")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running ip-link: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	ret := make([]InterfaceInfo, 0)
	r := regexp.MustCompile("^\\d+: +([^:]+): +<([^>]+)>.* mtu +(\\d+) .*link/ether +([^ ]+)")
	for _, line := range lines {
		netInterface := InterfaceInfo{}
		matches := r.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}
		netInterface.Name = matches[1]
		netInterface.State = matches[2]
		netInterface.Mtu, _ = strconv.Atoi(matches[3])
		netInterface.Mac = matches[4]
		ret = append(ret, netInterface)
	}
	return ret
}
