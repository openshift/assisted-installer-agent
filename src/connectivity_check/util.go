package connectivity_check

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

//go:generate mockery --name Executer --inpackage
type Executer interface {
	Execute(command string, args ...string) (string, error)
}

type executer struct{}

func (e *executer) Execute(command string, args ...string) (ret string, err error) {
	b, err := exec.Command(command, args...).CombinedOutput()
	if b != nil {
		ret = string(b)
	}
	return
}

func newExecuter() Executer {
	return &executer{}
}

func getOutgoingNics(dryRunConfig *config.DryRunConfig, d util.IDependencies) []string {
	ret := make([]string, 0)
	if d == nil {
		d = util.NewDependencies(dryRunConfig, "")
	}
	interfaces, err := d.Interfaces()
	if err != nil {
		log.WithError(err).Warnf("Get outgoing nics")
		return nil
	}
	for _, intf := range interfaces {
		if !(intf.IsPhysical() || intf.IsBonding() || intf.IsVlan()) {
			continue
		}

		// In order to remediate issue with polluting ARP table by using enslaved interfaces
		// (https://bugzilla.redhat.com/show_bug.cgi?id=2105358) we are only using as outgoing
		// NICs those interfaces that have at least one IP address assigned.
		addrs, _ := intf.Addrs()
		if len(addrs) == 0 {
			log.Infof("Skipping NIC %s (MAC %s) because of no addresses", intf.Name(), intf.HardwareAddr().String())
			continue
		}
		ret = append(ret, intf.Name())
	}
	return ret
}

func getIPAddressFromCIDR(cidr string) string {
	parts := strings.Split(cidr, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func getAllMacAddresses(checkHost *models.ConnectivityCheckHost) []string {
	var ret []string
	for _, nic := range checkHost.Nics {
		ret = append(ret, nic.Mac.String())
	}
	return ret
}

func macInDstMacs(mac string, allDstMACs []string) bool {
	for _, dstMAC := range allDstMACs {
		if strings.EqualFold(mac, dstMAC) {
			return true
		}
	}
	return false
}

func regexMatchFor(regex, line string) ([]string, error) {
	r := regexp.MustCompile(regex)
	p := r.FindStringSubmatch(line)
	if len(p) < 2 {
		return nil, errors.Errorf("unable to parse %s with regex %s", line, regex)
	}
	return p, nil
}
