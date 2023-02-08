package ui

import (
	"github.com/openshift/agent-installer-utils/tools/agent_tui/newt"
	"github.com/rivo/tview"
)

const (
	PAGE_SPLASHSCREEN string = "splashscreen"
)

func (u *UI) createSplashScreen() {
	u.splashScreen = tview.NewModal()
	u.splashScreen.SetBackgroundColor(newt.ColorBlack).
		SetBorderColor(newt.ColorBlack).
		SetBorder(true)
	u.splashScreen.SetText("Please wait, collecting initial check results")
	u.pages.AddPage(PAGE_SPLASHSCREEN, u.splashScreen, true, false)
}

func (u *UI) ShowSplashScreen() {
	u.app.SetFocus(u.splashScreen)
	u.pages.ShowPage(PAGE_SPLASHSCREEN)
}

func (u *UI) HideSplashScreen() {
	u.pages.HidePage(PAGE_SPLASHSCREEN)
}

func (u *UI) IsSplashScreenActive() bool {
	return u.splashScreen.HasFocus()
}
