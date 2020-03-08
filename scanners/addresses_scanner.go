package scanners

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strings"
)

func ReadAddresses() map[string][]string {
	cmd := exec.Command("ip", "-o",  "address", "list")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running ip-address: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	ret := make(map[string][]string)
	r := regexp.MustCompile("^\\d+: +([^ ]+) +.*inet +([^ ]+)")
	for _, line := range lines {
		matches := r.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		interfaceName := matches[1]
		address := matches[2]
		addresses, ok := ret[interfaceName]
		if !ok {
			addresses = make([]string, 0)
		}
		ret[interfaceName] = append(addresses, address)
	}
	return ret
}
