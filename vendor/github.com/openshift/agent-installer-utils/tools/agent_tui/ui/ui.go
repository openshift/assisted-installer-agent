package ui

import (
	"sync/atomic"

	"github.com/openshift/agent-installer-utils/tools/agent_tui/checks"
	"github.com/rivo/tview"
)

type UI struct {
	app                 *tview.Application
	pages               *tview.Pages
	grid                *tview.Grid // layout for the checks page
	primaryCheck        *tview.Table
	checks              *tview.Table    // summary of all checks
	details             *tview.TextView // where errors from checks are displayed
	form                *tview.Form     // contains "Configure network" button
	timeoutModal        *tview.Modal    // popup window that times out
	splashScreen        *tview.Modal    // display initial waiting message
	nmtuiActive         atomic.Value
	timeoutDialogActive atomic.Value
	timeoutDialogCancel chan bool
}

func NewUI(app *tview.Application, config checks.Config) *UI {
	ui := &UI{
		app:                 app,
		timeoutDialogCancel: make(chan bool),
	}
	ui.nmtuiActive.Store(false)
	ui.timeoutDialogActive.Store(false)
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

func (u *UI) create(config checks.Config) {
	u.pages = tview.NewPages()
	u.createCheckPage(config)
	u.createTimeoutModal(config)
	u.createSplashScreen()
}
