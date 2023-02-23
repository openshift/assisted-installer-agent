package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/net"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/newt"
	"github.com/rivo/tview"
)

func ModalNetStateJSONPage(ns *net.NetState, pages *tview.Pages) (*tview.Modal, error) {
	if pages == nil {
		return nil, fmt.Errorf("can't add modal NetState page to nil pages")
	}

	modal := tview.NewModal().
		SetText(fmt.Sprintf("%+v", *ns)).
		SetTextColor(tcell.ColorBlack).
		SetBackgroundColor(newt.ColorGray)
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyESC {
			pages.HidePage("netstate")
		}
		return event
	})

	return modal, nil
}
