package connectivity_check

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const ipv4LocalLinkCIDR = "169.254.0.0/16"
const ipv6LocalLinkCIDR = "fe80::/10"

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

func analyzeAddress(addr net.Addr) (isIpv4 bool, isLinkLocal bool, err error) {
	ipNet, ok := addr.(*net.IPNet)
	if !ok {
		return false, false, fmt.Errorf("could not cast to *net.IPNet")
	}
	_, bits := ipNet.Mask.Size()
	isIpv4 = bits == 32
	var linkLocalNet *net.IPNet
	if isIpv4 {
		_, linkLocalNet, err = net.ParseCIDR(ipv4LocalLinkCIDR)
	} else {
		_, linkLocalNet, err = net.ParseCIDR(ipv6LocalLinkCIDR)
	}
	isLinkLocal = linkLocalNet.Contains(ipNet.IP)
	return
}

func getOutgoingNics(dryRunConfig *config.DryRunConfig, d util.IDependencies) []OutgoingNic {
	ret := make([]OutgoingNic, 0)
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

		outgoingNic := OutgoingNic{Name: intf.Name(), MTU: intf.MTU()}
		for _, addr := range addrs {
			isIpv4, isLinkLocal, err := analyzeAddress(addr)
			if err != nil {
				log.Warnf("failed analizing address %s", addr.String())
				continue
			}
			if isLinkLocal {
				continue
			}
			if isIpv4 {
				outgoingNic.HasIpv4Addresses = true
			} else {
				outgoingNic.HasIpv6Addresses = true
			}
		}
		if !outgoingNic.HasIpv4Addresses && !outgoingNic.HasIpv6Addresses {
			log.Infof("Skipping NIC %s (MAC %s) because of no valid addresses", intf.Name(), intf.HardwareAddr().String())
			continue
		}
		outgoingNic.Addresses = addrs
		ret = append(ret, outgoingNic)
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
