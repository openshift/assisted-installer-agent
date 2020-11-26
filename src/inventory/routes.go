package inventory

import (
	"net"

	"github.com/sirupsen/logrus"

	"github.com/openshift/assisted-service/models"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type routeFinder struct {
	dependencies IDependencies
}

func newRouteFinder(dependencies IDependencies) *routeFinder {
	return &routeFinder{dependencies:dependencies}
}


// usableIPv6Route returns true if the passed route is acceptable for AddressesRouting
func usableIPv6Route(route netlink.Route) bool {
	// Ignore default routes
	if route.Dst == nil {
		return false
	}
	// Ignore non-IPv6 routes
	if net.IPv6len != len(route.Dst.IP) {
		return false
	}
	// Ignore non-advertised routes
	if route.Protocol != unix.RTPROT_RA {
		return false
	}

	return true
}

func (r *routeFinder) getLinkV6Routes(linkName string) (routeList []netlink.Route, err error) {
	link, err := r.dependencies.LinkByName(linkName)
	if err != nil {
		return nil, err
	}
	routes, err := r.dependencies.RouteList(link, netlink.FAMILY_V6)
	if err != nil {
		return nil, err
	}

	routeList = make([]netlink.Route, 0)
	for _, route := range routes {
		if !usableIPv6Route(route) {
			logrus.Debugf("Ignoring filtered route %+v", route)
			continue
		}
		routeList = append(routeList, route)
	}

	return routeList, nil
}

func setV6PrefixesForAddresses(interfaces []*models.Interface, dependencies IDependencies) {
	finder := newRouteFinder(dependencies)
	for _, intf := range interfaces {
		if len(intf.IPV6Addresses) == 0 {
			continue
		}
		routes, err := finder.getLinkV6Routes(intf.Name)
		if err != nil {
			logrus.WithError(err).Warnf("Could not get routes for interface %s", intf.Name)
			continue
		}
		for i, addr := range intf.IPV6Addresses {
			ip, _, err := net.ParseCIDR(addr)
			if err != nil {
				logrus.WithError(err).Warnf("Could not parse CIDR %s", addr)
				continue
			}
			for _, route := range routes {
				containmentNet := net.IPNet{IP: route.Dst.IP, Mask: route.Dst.Mask}
				if containmentNet.Contains(ip) {
					intf.IPV6Addresses[i] = (&net.IPNet{IP: ip, Mask: route.Dst.Mask}).String()
					break
				}
			}
		}
	}
}
