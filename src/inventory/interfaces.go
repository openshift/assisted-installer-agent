package inventory

import (
	"fmt"
	"net"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

const ipv6LocalLinkCIDR = "fe80::/10"

type interfaces struct {
	dependencies util.IDependencies
}

func newInterfaces(dependencies util.IDependencies) *interfaces {
	return &interfaces{dependencies: dependencies}
}

func (i *interfaces) hasCarrier(name string) bool {
	fname := fmt.Sprintf("/sys/class/net/%s/carrier", name)
	b, err := i.dependencies.ReadFile(fname)
	if err != nil {
		logrus.WithError(err).Debugf("Reading file %s", fname)
		return false
	}
	return strings.TrimSpace(string(b)) == "1"
}

func (i *interfaces) getDeviceField(name, field string) string {
	fname := fmt.Sprintf("/sys/class/net/%s/device/%s", name, field)
	b, err := i.dependencies.ReadFile(fname)
	if err != nil {
		logrus.WithError(err).Debugf("Reading file %s", fname)
		return ""
	}
	return strings.TrimSpace(string(b))

}

func ipWithCidrInCidr(ipWithCidrStr, cidrStr string) bool {
	ip, _, err := net.ParseCIDR(ipWithCidrStr)
	if ip == nil || err != nil {
		return false
	}
	_, ipnet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return false
	}
	return ipnet.Contains(ip)
}

func analyzeAddress(addr net.Addr) (isIpv4 bool, addrStr string, err error) {
	ipNet, ok := addr.(*net.IPNet)
	if !ok {
		return false, "", fmt.Errorf("Could not cast to *net.IPNet")
	}
	mask, _ := ipNet.Mask.Size()
	addrStr = fmt.Sprintf("%s/%d", ipNet.IP.String(), mask)
	isIpv4 = strings.Contains(addrStr, ".")
	return
}

func (i *interfaces) getBiosDevname(name string) string {
	o, e, exitCode := i.dependencies.Execute("biosdevname", "-i", name)
	if exitCode != 0 {
		logrus.Debugf("biosdevname error: %s", e)
	}
	return strings.TrimSpace(o)
}

func getFlags(flags net.Flags) []string {
	flagsStr := flags.String()
	if flagsStr == "0" {
		return make([]string, 0)
	} else {
		return strings.Split(flagsStr, "|")
	}
}

func (i *interfaces) getInterfaces() []*models.Interface {
	ret := make([]*models.Interface, 0)
	ins, err := i.dependencies.Interfaces()
	if err != nil {
		logrus.WithError(err).Warnf("Retrieving interfaces")
		return ret
	}
	for _, in := range ins {
		if !(in.IsPhysical() || in.IsBonding() || in.IsVlan()) {
			continue
		}
		rec := models.Interface{
			HasCarrier:    i.hasCarrier(in.Name()),
			IPV4Addresses: make([]string, 0),
			IPV6Addresses: make([]string, 0),
			MacAddress:    in.HardwareAddr().String(),
			Name:          in.Name(),
			Mtu:           int64(in.MTU()),
			Biosdevname:   i.getBiosDevname(in.Name()),
			Product:       i.getDeviceField(in.Name(), "device"),
			Vendor:        i.getDeviceField(in.Name(), "vendor"),
			Flags:         getFlags(in.Flags()),
			SpeedMbps:     in.SpeedMbps(),
		}
		addrs, err := in.Addrs()
		if err != nil {
			logrus.WithError(err).Warnf("Retrieving addresses for %s", in.Name())
			continue
		}
		for _, addr := range addrs {
			isIPv4, addrStr, err := analyzeAddress(addr)
			if err != nil {
				logrus.WithError(err).Warnf("While analyzing addr")
				continue
			}
			if isIPv4 {
				rec.IPV4Addresses = append(rec.IPV4Addresses, addrStr)
			} else if !ipWithCidrInCidr(addrStr, ipv6LocalLinkCIDR) {
				rec.IPV6Addresses = append(rec.IPV6Addresses, addrStr)
			}
		}
		ret = append(ret, &rec)
	}
	setV6PrefixesForAddresses(ret, i.dependencies)
	return ret
}

func GetInterfaces(dependencies util.IDependencies) []*models.Interface {
	return newInterfaces(dependencies).getInterfaces()
}

func setV6PrefixesForAddresses(interfaces []*models.Interface, dependencies util.IDependencies) {
	for _, intf := range interfaces {
		if len(intf.IPV6Addresses) == 0 {
			continue
		}

		if err := util.SetV6PrefixesForAddress(intf.Name, dependencies, logrus.StandardLogger(), intf.IPV6Addresses); err != nil {
			logrus.WithError(err).Warnf("Failed to set V6 prefix for interface %s address %s", intf.Name, intf.IPV6Addresses)
		}
	}
}
