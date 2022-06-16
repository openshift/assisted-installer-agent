package util

import (
	netlink "github.com/vishvananda/netlink"
)

//go:generate mockery -name Link -inpkg
type Link interface {
	netlink.Link
}
