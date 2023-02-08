package ui

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/nmstate/nmstate/rust/src/go/nmstate/v2"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/dialogs"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/net"
	"github.com/rivo/tview"
)

func (u *UI) ShowNMTUI(treeView *tview.TreeView) {
	u.nmtuiActive.Store(true)
	defer u.nmtuiActive.Store(false)

	u.app.Suspend(func() {
		cmd := exec.Command("nmtui")
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			dialogs.PanicDialog(u.app, err)
		}
	})
	nm := nmstate.New()
	state, err := nm.RetrieveNetState()
	if err != nil {
		dialogs.PanicDialog(u.app, err)
	}

	var netState net.NetState
	if err := json.Unmarshal([]byte(state), &netState); err != nil {
		dialogs.PanicDialog(u.app, err)
	}

	//netStatePage, err := modalNetStateJSONPage(&netState, pages)
	if treeView == nil {
		netStatePage, err := u.ModalTreeView(netState)
		if err != nil {
			dialogs.PanicDialog(u.app, err)
		}
		u.pages.AddPage("netstate", netStatePage, true, true)
	} else {
		updatedTreeView, err := u.TreeView(netState)
		if err != nil {
			dialogs.PanicDialog(u.app, err)
		}
		treeView.SetRoot(updatedTreeView.GetRoot())
	}
}
