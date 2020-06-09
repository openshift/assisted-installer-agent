package scanners

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/filanov/bm-inventory/models"
	log "github.com/sirupsen/logrus"
)

func ReadAddresses() map[string][]*models.Cidr {
	cmd := exec.Command("ip", "-o", "address", "list")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running ip-address: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	ret := make(map[string][]*models.Cidr)
	r := regexp.MustCompile("^\\d+: +([^ ]+) +.*inet +([0-9.]+)/(\\d+)")
	for _, line := range lines {
		matches := r.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}
		interfaceName := matches[1]
		address := matches[2]
		mask, _ := strconv.ParseInt(matches[3], 10, 64)
		cidrs, ok := ret[interfaceName]
		if !ok {
			cidrs = make([]*models.Cidr, 0)
		}
		ret[interfaceName] = append(cidrs, &models.Cidr{
			IPAddress: address,
			Mask:      mask,
		})
	}
	return ret
}
