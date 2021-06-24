package inventory

import (
	"net"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const (
	familyIPv4 int = 4
	familyIPv6 int = 6
)

type handler interface {
	getRouteList() ([]netlink.Route, error)
	getLinkName(linkIndex int) (string, error)
	getFamily() int
}

type routeHandler struct {
	family int
}

func (rh routeHandler) getRouteList() ([]netlink.Route, error) {
	return netlink.RouteList(nil, rh.family)
}

func (rh routeHandler) getLinkName(linkIndex int) (string, error) {
	link, err := netlink.LinkByIndex(linkIndex)
	if err != nil {
		return "", err
	}
	return link.Attrs().Name, nil
}

func (rh routeHandler) getFamily() int {
	return rh.family
}

func getIPZero(family int) *net.IP {
	if family == familyIPv4 {
		return &net.IPv4zero
	}
	return &net.IPv6zero
}

func GetRoutes(dependencies util.IDependencies) []*models.Route {

	rh4 := routeHandler{family: familyIPv4}
	rh6 := routeHandler{family: familyIPv6}
	routes, err := getIPRoutes(rh4)
	if err != nil {
		logrus.Errorf("Unable to determine the IPv4 routes: %s", err)
		return []*models.Route{}
	}
	ipv6Routes, err := getIPRoutes(rh6)
	if err != nil {
		logrus.Errorf("Unable to determine the IPv6 routes: %s", err)
		return routes //If ipv6 failed, we still return the ipv4 default route(s) since that could be sufficient.
	}
	routes = append(routes, ipv6Routes...)
	return routes
}

func getIPRoutes(h handler) ([]*models.Route, error) {
	rList, err := h.getRouteList()
	if err != nil {
		logrus.Errorf("Unable to retrieve the IPv%d routes: %s", h.getFamily(), err)
		return []*models.Route{}, err
	}
	routes := []*models.Route{}
	for _, r := range rList {
		linkName, err := h.getLinkName(r.LinkIndex)
		if err != nil {
			logrus.Errorf("Unable to retrieve the link name for index %d: %s", r.LinkIndex, err)
			return nil, err
		}
		var dst string
		if r.Dst == nil {
			dst = getIPZero(h.getFamily()).String()
		} else {
			dst = r.Dst.IP.String()
		}
		routes = append(routes, &models.Route{
			Interface:   linkName,
			Destination: dst,
			Gateway:     r.Gw.String(),
			Family:      int32(h.getFamily()),
		})
	}
	return routes, nil
}
