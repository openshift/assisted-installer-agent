package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/net"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/newt"
	"github.com/rivo/tview"
)

func getIfaceTree(iface net.Iface) *tview.TreeNode {
	root := tview.NewTreeNode(fmt.Sprintf("%s (%s)", iface.Name, iface.Type)).SetColor(tcell.ColorBlack)
	root.AddChild(tview.NewTreeNode(fmt.Sprintf("MTU: %d", iface.MTU)).SetColor(tcell.ColorBlack))
	root.AddChild(tview.NewTreeNode(fmt.Sprintf("State: %s", iface.State)).SetColor(tcell.ColorBlack))

	if len(iface.IPv4.Addresses) > 0 {
		IPv4Node := tview.NewTreeNode("IPv4 Addresses").SetColor(tcell.ColorBlack)
		for _, address := range iface.IPv4.Addresses {
			IPv4Node.AddChild(tview.NewTreeNode(address.String()).SetColor(tcell.ColorBlack))
		}
		root.AddChild(IPv4Node)
	}
	if len(iface.IPv6.Addresses) > 0 {
		IPv6Node := tview.NewTreeNode("IPv6 Addresses").SetColor(tcell.ColorBlack)
		for _, address := range iface.IPv6.Addresses {
			IPv6Node.AddChild(tview.NewTreeNode(address.String()).SetColor(tcell.ColorBlack))
		}
		root.AddChild(IPv6Node)
	}

	return root
}

func getRouteTree(route net.Route) *tview.TreeNode {
	var dest string
	if net.IsIPv4DefaultRoute(route.Destination) || net.IsIPv6DefaultRoute(route.Destination) {
		dest = "default"
	} else {
		dest = route.Destination
	}

	root := tview.NewTreeNode(dest).SetColor(tcell.ColorBlack)
	root.AddChild(tview.NewTreeNode(fmt.Sprintf("Next hop address: %s", route.NextHopAddr)).SetColor(tcell.ColorBlack))
	root.AddChild(tview.NewTreeNode(fmt.Sprintf("Next hop interface: %s", route.NextHopIface)).SetColor(tcell.ColorBlack))
	return root
}

func (u *UI) ModalTreeView(netState net.NetState) (tview.Primitive, error) {
	if u.pages == nil {
		return nil, fmt.Errorf("can't make a NetState treeView page for nil pages")
	}

	treeView, err := u.TreeView(netState)
	if err != nil {
		return nil, err
	}
	width := 40
	height := 40
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(treeView, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false), err
}

func (u *UI) TreeView(netState net.NetState) (*tview.TreeView, error) {
	if u.pages == nil {
		return nil, fmt.Errorf("can't make a NetState treeView page for nil pages")
	}

	root := tview.NewTreeNode(fmt.Sprintf("[black::b]%s", netState.Hostname.Running))
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root).SetDoneFunc(
		func(key tcell.Key) {
			u.pages.RemovePage("netstate")
			u.returnFocusToChecks()
		})

	tree.SetTitle("Network Status").
		SetBackgroundColor(newt.ColorGray).
		SetBorder(true).
		SetBorderColor(tcell.ColorBlack).
		SetTitleColor(tcell.ColorBlack)

	tree.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			if event.Rune() == 'q' {
				u.pages.RemovePage("netstate")
				u.returnFocusToChecks()
			}
			return event
		})

	interfaces := tview.NewTreeNode("Interfaces").SetColor(tcell.ColorBlack)
	root.AddChild(interfaces)

	defaultIface, err := netState.GetDefaultNextHopIface()
	if err != nil {
		return nil, fmt.Errorf("failed to generate network state view: %w", err)
	}
	if defaultIface != nil {
		interfaces.AddChild(getIfaceTree(*defaultIface).SetColor(tcell.ColorGreen))
	}
	for _, iface := range netState.Ifaces {
		if defaultIface != nil && defaultIface.Name == iface.Name {
			continue // Skip defaultRouteIface, since we always display it first
		}
		interfaces.AddChild(getIfaceTree(iface))
	}

	if len(netState.Routes.Running) > 0 {
		routes := tview.NewTreeNode("Routes").SetColor(tcell.ColorBlack)
		root.AddChild(routes)

		for _, route := range netState.Routes.Running {
			routes.AddChild(getRouteTree(route))
		}
	}

	var dns *tview.TreeNode
	if len(netState.DNS.Running.Servers) > 0 {
		dns = tview.NewTreeNode("DNS").SetColor(tcell.ColorBlack)
		root.AddChild(dns)

		servers := tview.NewTreeNode("Servers").SetColor(tcell.ColorBlack)
		dns.AddChild(servers)
		for _, server := range netState.DNS.Running.Servers {
			servers.AddChild(tview.NewTreeNode(server).SetColor(tcell.ColorBlack))
		}
	}

	if len(netState.DNS.Running.SearchDomains) > 0 {
		if dns == nil {
			dns = tview.NewTreeNode("DNS").SetColor(tcell.ColorBlack)
			root.AddChild(dns)
		}
		searchDomains := tview.NewTreeNode("Search domains").SetColor(tcell.ColorBlack)
		dns.AddChild(searchDomains)

		for _, search := range netState.DNS.Running.SearchDomains {
			searchDomains.AddChild(tview.NewTreeNode(search).SetColor(tcell.ColorBlack))
		}
	}

	return tree, nil
}
