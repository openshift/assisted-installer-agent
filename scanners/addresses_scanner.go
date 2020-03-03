package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strings"
)

type AddressInfo struct {
	InterfaceName string  `json:"interface_name"`
	Address string  `json:"address"`
}

func ReadAddresses() [] AddressInfo {
	cmd := exec.Command("ip", "-o",  "address", "list")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running ip-address: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	ret := make([]AddressInfo, 0)
	r := regexp.MustCompile("^\\d+: +([^ ]+) +.*inet +([^ ]+)")
	for _, line := range lines {
		netInterface := AddressInfo{}
		matches := r.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		netInterface.InterfaceName = matches[1]
		netInterface.Address = matches[2]
		ret = append(ret, netInterface)
	}
	return ret
}
