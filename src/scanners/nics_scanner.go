package scanners

import (
	"github.com/filanov/bm-inventory/models"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func ReadNics() []*models.Nic {
	cmd := exec.Command("ip", "-o",  "link", "list")
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("Error running ip-link: %s", err.Error())
		return nil
	}
	lines := strings.Split(string(bytes), "\n")
	ret := make([]*models.Nic, 0)
	cidrs := ReadAddresses()
	r := regexp.MustCompile("^\\d+: +([^:]+): +<([^>]+)>.* mtu +(\\d+) .*link/ether +([^ ]+)")
	for _, line := range lines {
		nic := &models.Nic{}
		matches := r.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}
		nic.Name = matches[1]
		nic.State = matches[2]
		nic.Mtu, _ = strconv.ParseInt(matches[3], 10, 64)
		nic.Mac = matches[4]
		nic.Cidrs, _ = cidrs[nic.Name]
		ret = append(ret, nic)
	}
	return ret
}
