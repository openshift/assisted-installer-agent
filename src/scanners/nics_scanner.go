package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type NicInfo struct {
	Name string `json:"name"`
	State string `json:"state"`
	Mtu int `json:"mtu"`
	Mac string `json:"mac"`
	IPAddresses []string `json:"ip_addresses,omitempty"`
}

func ReadNics() []NicInfo {
	cmd := exec.Command("ip", "-o",  "link", "list")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running ip-link: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	ret := make([]NicInfo, 0)
	addresses := ReadAddresses()
	r := regexp.MustCompile("^\\d+: +([^:]+): +<([^>]+)>.* mtu +(\\d+) .*link/ether +([^ ]+)")
	for _, line := range lines {
		nic := NicInfo{}
		matches := r.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}
		nic.Name = matches[1]
		nic.State = matches[2]
		nic.Mtu, _ = strconv.Atoi(matches[3])
		nic.Mac = matches[4]
		nic.IPAddresses, _ = addresses[nic.Name]
		ret = append(ret, nic)
	}
	return ret
}
