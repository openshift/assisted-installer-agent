package ui

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/nmstate/nmstate/rust/src/go/nmstate/v2"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/net"
)

func (u *UI) ShowNMTUI() error {
	u.nmtuiActive.Store(true)
	defer u.nmtuiActive.Store(false)

	var nmtuiErr error
	u.app.Suspend(func() {
		cmd := exec.Command("nmtui")
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		nmtuiErr = cmd.Run()
	})
	if nmtuiErr != nil {
		return nmtuiErr
	}

	nm := nmstate.New()
	state, err := nm.RetrieveNetState()
	if err != nil {
		return err
	}

	var netState net.NetState
	if err := json.Unmarshal([]byte(state), &netState); err != nil {
		return err
	}

	netStatePage, err := u.ModalTreeView(netState)
	if err != nil {
		return err
	}
	u.pages.AddPage("netstate", netStatePage, true, true)

	return nil
}
