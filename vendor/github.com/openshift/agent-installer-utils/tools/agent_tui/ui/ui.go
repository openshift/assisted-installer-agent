package ui

import (
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/checks"
	"github.com/rivo/tview"
)

type UI struct {
	app                 *tview.Application
	pages               *tview.Pages
	mainFlex, innerFlex *tview.Flex
	primaryCheck        *tview.Table
	checks              *tview.Table    // summary of all checks
	details             *tview.TextView // where errors from checks are displayed
	form                *tview.Form     // contains "Configure network" button
	timeoutModal        *tview.Modal    // popup window that times out
	splashScreen        *tview.Modal    // display initial waiting message
	nmtuiActive         atomic.Value
	timeoutDialogActive atomic.Value
	timeoutDialogCancel chan bool
	dirty               atomic.Value // dirty flag set if the user interacts with the ui

	focusableItems []tview.Primitive // the list of widgets that can be focused
	focusedItem    int               // the current focused widget
}

func NewUI(app *tview.Application, config checks.Config) *UI {
	ui := &UI{
		app:                 app,
		timeoutDialogCancel: make(chan bool),
	}
	ui.nmtuiActive.Store(false)
	ui.timeoutDialogActive.Store(false)
	ui.dirty.Store(false)
	ui.create(config)
	return ui
}

func (u *UI) GetApp() *tview.Application {
	return u.app
}

func (u *UI) GetPages() *tview.Pages {
	return u.pages
}

func (u *UI) returnFocusToChecks() {
	u.pages.SwitchToPage(PAGE_CHECKSCREEN)
	// shifting focus back to the "Configure network"
	// button requires setting focus in this sequence
	// form -> form-button
	u.app.SetFocus(u.form)
	u.app.SetFocus(u.form.GetButton(0))
}

func (u *UI) IsNMTuiActive() bool {
	return u.nmtuiActive.Load().(bool)
}

func (u *UI) setIsTimeoutDialogActive(isActive bool) {
	u.timeoutDialogActive.Store(isActive)
}

func (u *UI) IsTimeoutDialogActive() bool {
	return u.timeoutDialogActive.Load().(bool)
}

func (u *UI) IsDirty() bool {
	return u.dirty.Load().(bool)
}

func (u *UI) create(config checks.Config) {
	u.pages = tview.NewPages()
	u.createCheckPage(config)
	u.createTimeoutModal(config)
	u.createSplashScreen()
	u.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		u.dirty.Store(true)
		return event
	})
}
