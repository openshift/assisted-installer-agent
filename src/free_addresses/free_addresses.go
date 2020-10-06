package free_addresses

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/pkg/errors"

	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

const (
	AddressLimit      = 8000
	MinSubnetMaskSize = 22
)

//go:generate mockery -name Executer -inpkg
type Executer interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
}

type ProcessExecuter struct{}

func (e *ProcessExecuter) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute(command, args...)
}

type status struct {
	State  string `xml:"state,attr"`
	Reason string `xml:"reason,attr"`
}

type address struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type host struct {
	Status    status     `xml:"status"`
	Addresses []*address `xml:"address"`
}

type nmaprun struct {
	XMLName xml.Name `xml:"nmaprun"`
	Hosts   []*host  `xml:"host"`
}

//  http://play.golang.org/p/m8TNTtygK0
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

/*
 * This function increases the ip by mask size.
 * If ip is IPv4 and mask is 32 -> the ip is increased by 1
 * if the mask is 24 -> the ip is increased by 256
 * In general assuming IPv4, the IP is increased by 2^(32-mask)
 */
func incSubnet(ip net.IP, mask int) {
	if mask == 0 {
		return
	}
	start := (mask - 1) / 8
	add := byte(1 << (7 - ((mask - 1) % 8)))
	prev := ip[start]
	ip[start] += add
	if ip[start] < prev {
		for j := start - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}
}

type scanner struct {
	exe Executer
	log logrus.FieldLogger
}

func newScanner(exe Executer, log logrus.FieldLogger) *scanner {
	return &scanner{
		exe: exe,
		log: log,
	}
}

func (s *scanner) scanSubNetwork(subNetwork string) ([]strfmt.IPv4, string, int) {
	ip, cidr, err := net.ParseCIDR(subNetwork)
	if err != nil {
		wrapped := errors.Wrapf(err, "Network %s is not a valid CIDR ", subNetwork)
		s.log.WithError(wrapped).Warn("ParseCIDR")
		return nil, wrapped.Error(), -1
	}
	if cidr.String() != subNetwork {
		stderr := fmt.Sprintf("Requested CIDR %s is not equal to provided subNetwork %s", cidr.String(), subNetwork)
		s.log.Warn(stderr)
		return nil, stderr, -1
	}
	o, e, exitCode := s.exe.Execute("nmap", "-sn", "-PR", "-n", "-oX", "-", subNetwork)
	if exitCode != 0 {
		s.log.Warnf("nmap failed with exit-code %d: %s", exitCode, e)
		return nil, e, exitCode
	}
	var nmaprun nmaprun
	err = xml.Unmarshal([]byte(o), &nmaprun)
	if err != nil {
		s.log.WithError(err).Warn("XML Unmarshal")
		return nil, err.Error(), -1
	}
	occupiedAddesses := make(map[string]int)
	for _, h := range nmaprun.Hosts {
		if h.Status.State == "up" {
			for _, a := range h.Addresses {
				if a.AddrType == "ipv4" {
					occupiedAddesses[a.Addr] = 0
					break
				}
			}
		}
	}
	ret := make([]strfmt.IPv4, 0)
	for ip = ip.To4(); cidr.Contains(ip); inc(ip) {
		_, ok := occupiedAddesses[ip.String()]
		if !ok {
			ret = append(ret, strfmt.IPv4(ip.String()))
		}
	}
	return ret, "", 0
}

func (s *scanner) scanNetwork(network string) (*models.FreeNetworkAddresses, string, int) {
	ret := models.FreeNetworkAddresses{
		Network: network,
	}
	ip, cidr, err := net.ParseCIDR(network)
	if err != nil {
		wrapped := errors.Wrapf(err, "Network %s is not a valid CIDR", network)
		s.log.WithError(wrapped).Warn("ParseCIDR")
		return &ret, wrapped.Error(), -1
	}
	if cidr.String() != network {
		stderr := fmt.Sprintf("Requested CIDR %s is not equal to provided network %s", cidr.String(), network)
		s.log.Warn(stderr)
		return &ret, stderr, -1
	}

	// ones is the mask of the subnet.  For example 1.2.3.0/24 -> ones == 24
	ones, _ := cidr.Mask.Size()

	var mask int

	/*
	 * The following logic works on the mask size (Example 1.2.3.0/24 - 24 is the mask size).
	 * We want to avoid sending subnets with too many addresses to nmap.  Therefore, we divide them to smaller segments.
	 * One important note: When talking about mask sizes, the smaller the mask size -> the larger the subnet.
	 * For example: for IPv4, subnet with mask /32 is a subnet with a single address. /31 contains 2 addresses.
	 * Generally speaking - the number of addresses in a subnet is 2^(32-mask).
	 * The way we verify that the subnet is not too large is by setting the mask as MAX(ones, MinSubnetMaskSize).
	 */
	if ones < MinSubnetMaskSize {
		mask = MinSubnetMaskSize
	} else {
		mask = ones
	}
	for ip = ip.To4(); cidr.Contains(ip) && len(ret.FreeAddresses) < AddressLimit; incSubnet(ip, mask) {
		subNetwork := fmt.Sprintf("%s/%d", ip.To4().String(), mask)
		result, e, exitCode := s.scanSubNetwork(subNetwork)
		if exitCode != 0 {
			return &ret, e, exitCode
		}
		ret.FreeAddresses = append(ret.FreeAddresses, result...)
	}
	return &ret, "", 0
}

func GetFreeAddresses(freeAddressesRequestStr string, e Executer, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var freeAddressesRequest models.FreeAddressesRequest
	err := json.Unmarshal([]byte(freeAddressesRequestStr), &freeAddressesRequest)
	if err != nil {
		log.WithError(err).Errorf("FreeAddresses: json.Unmarshal")
		return "", err.Error(), -1
	}
	freeAddresses := models.FreeNetworksAddresses{}
	s := newScanner(e, log)
	for _, r := range freeAddressesRequest {
		rec, e, exitCode := s.scanNetwork(r)
		if exitCode != 0 {
			return "", e, exitCode
		}
		freeAddresses = append(freeAddresses, rec)
	}
	b, err := json.Marshal(&freeAddresses)
	if err != nil {
		log.WithError(err).Error("Free addresses marshal")
		return "", err.Error(), -1
	}
	return string(b), "", 0
}
