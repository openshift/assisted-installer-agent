package util

import(
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

//go:generate mockery -name Interface -inpkg
type Interface interface {
        MTU() int
        Name() string
        HardwareAddr() net.HardwareAddr
        Flags() net.Flags
        Addrs() ([]net.Addr, error)
        IsPhysical() bool
        IsBonding() bool
        IsVlan() bool
        SpeedMbps() int64
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
                logrus.WithError(err).Warnf("Could not determine if interface %s is physical", n.netInterface.Name)
                return true
        }
        return !strings.Contains(evaledPath, "/virtual/")
}

func (n *NetworkInterface) IsBonding() bool {
        link, err := n.dependencies.LinkByName(n.netInterface.Name)
        if err != nil {
                return false
        }
        return link.Type() == "bond"
}

func (n *NetworkInterface) IsVlan() bool {
        link, err := n.dependencies.LinkByName(n.netInterface.Name)
        if err != nil {
                return false
        }
        return link.Type() == "vlan"
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
