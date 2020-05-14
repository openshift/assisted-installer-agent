package inventory

import (
	"fmt"
	"github.com/filanov/bm-inventory/models"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
)


//go:generate mockery -name Interface -inpkg
type Interface interface {
	MTU()          int
	Name()         string
	HardwareAddr() net.HardwareAddr
	Flags()        net.Flags
	Addrs()        ([]net.Addr, error)
	IsPhysical()   bool
	SpeedMbps()    int64
}

type NetworkInterface struct {
	netInterface net.Interface
	dependencies IDependencies
}

func (n *NetworkInterface) MTU() int {
	return n.netInterface.MTU
}

func (n *NetworkInterface) Name() string {
	return n.netInterface.Name
}

func (n *NetworkInterface) HardwareAddr() net.HardwareAddr {
	return n.netInterface.HardwareAddr
}

func (n *NetworkInterface) Flags() net.Flags {
	return n.netInterface.Flags
}

func (n *NetworkInterface) Addrs() ([]net.Addr, error) {
	return n.netInterface.Addrs()
}

func (n *NetworkInterface) IsPhysical() bool {
	evaledPath, err := n.dependencies.EvalSymlinks(fmt.Sprintf("/sys/class/net/%s", n.netInterface.Name))
	if err != nil {
		logrus.WithError(err).Warnf("Could not determin if interface %s is physical", n.netInterface.Name)
		return true
	}
	return !strings.Contains(evaledPath, "/virtual/")
}

func (n *NetworkInterface) SpeedMbps() int64 {
	b, err := n.dependencies.ReadFile(fmt.Sprintf("/sys/class/net/%s/speed", n.Name()))
	if err != nil {
		logrus.WithError(err).Warnf("Could not read %s speed", n.Name())
		return 0
	}
	ret, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 32)
	if err != nil {
		logrus.WithError(err).Warnf("Could not parse %s speed", n.Name())
	}
	return ret
}

type interfaces struct {
	dependencies IDependencies
}

func newInterfaces(dependencies IDependencies) *interfaces {
	return &interfaces{dependencies:dependencies}
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

func analyzeAddress(addr net.Addr) (isIpv4 bool, addrStr string, err error) {
	ipNet, ok := addr.(*net.IPNet)
	if !ok {
		return false, "", fmt.Errorf("Could not cast to *net.IPNet")
	}
	mask, _ := ipNet.Mask.Size()
	addrStr = fmt.Sprintf("%s/%d",ipNet.IP.String(), mask)
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

func getFlags(flags net.Flags) [] string {
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
		if !in.IsPhysical() {
			continue
		}
		rec := models.Interface{
			HasCarrier:    i.hasCarrier(in.Name()),
			IPV4Addresses: make([]string, 0),
			IPV6Addresses: make([]string, 0),
			MacAddress:    in.HardwareAddr().String(),
			Name:          in.Name(),
			Mtu:		   int64(in.MTU()),
			Biosdevname:   i.getBiosDevname(in.Name()),
			Product:       i.getDeviceField(in.Name(), "device"),
			Vendor:        i.getDeviceField(in.Name(), "vendor"),
			Flags: 		   getFlags(in.Flags()),
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
			} else {
				rec.IPV6Addresses = append(rec.IPV6Addresses, addrStr)
			}
		}
		ret = append(ret, &rec)
	}
	return ret
}


func GetInterfaces(depenndecies IDependencies) []*models.Interface {
	return newInterfaces(depenndecies).getInterfaces()
}
