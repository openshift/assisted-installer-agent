package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/checks"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/newt"
	"github.com/rivo/tview"
)

const (
	CONFIGURE_NETWORK_LABEL string = "Configure Networking"
	RELEASE_IMAGE_LABEL     string = "Release Image"
	CONFIGURE_BUTTON        string = "<Configure network>"
	QUIT_BUTTON             string = "<Quit>"
	PAGE_CHECKSCREEN        string = "checkScreen"
)

func (u *UI) SetPullCheck(cr checks.CheckResult) {
	u.setCheck(u.primaryCheck, cr, 1, "release image pull error")
}

func (u *UI) SetDNSCheck(cr checks.CheckResult) {
	u.setCheck(u.checks, cr, 0, "nslookup failure")
}

func (u *UI) SetPingCheck(cr checks.CheckResult) {
	u.setCheck(u.checks, cr, 1, "ping failure")
}

func (u *UI) SetHttpGetCheck(cr checks.CheckResult) {
	u.setCheck(u.checks, cr, 2, "http server not responding")
}

func (u *UI) setCheck(table *tview.Table, cr checks.CheckResult, row int, msg string) {
	u.app.QueueUpdateDraw(func() {
		if cr.Success {
			u.markCheckSuccess(table, row, 0)
		} else {
			u.markCheckFail(table, row, 0)
			u.appendNewErrorToDetails(msg, cr.Details)
		}
	})
}

func (u *UI) markCheckSuccess(table *tview.Table, row int, col int) {
	table.SetCell(row, col, &tview.TableCell{
		Text:            " ✓",
		Color:           tcell.ColorLimeGreen,
		BackgroundColor: newt.ColorGray})
}

func (u *UI) markCheckFail(table *tview.Table, row int, col int) {
	table.SetCell(row, col, &tview.TableCell{
		Text:            " ✖",
		Color:           newt.ColorRed,
		BackgroundColor: newt.ColorGray})
}

func (u *UI) markCheckUnknown(table *tview.Table, row int, col int) {
	table.SetCell(row, col, &tview.TableCell{
		Text:            " ?",
		Color:           newt.ColorBlack,
		BackgroundColor: newt.ColorGray})
}

func (u *UI) setCheckDescription(table *tview.Table, row int, col int, description string) {
	table.SetCell(row, col, &tview.TableCell{
		Text:            description,
		Color:           newt.ColorBlack,
		BackgroundColor: newt.ColorGray})
}

func (u *UI) appendNewErrorToDetails(heading string, errorString string) {
	u.appendToDetails(fmt.Sprintf("%s%s:%s\n%s", "[red]", heading, "[black]", errorString))
}

func (u *UI) appendToDetails(newLines string) {
	current := u.details.GetText(false)
	if len(current) > 10000 {
		// if details run more than 10000 characters, reset
		current = ""
	}
	u.details.SetText(current + newLines)
}

func (u *UI) setCheckWidget(table *tview.Table, row int, checkType string, desc string, config checks.Config) {
	u.markCheckUnknown(table, row, 0)

	checkDesc := desc
	if strings.Contains(desc, "%s") {
		checkDesc = fmt.Sprintf(desc, config.ReleaseImageHostname)
	}
	u.setCheckDescription(table, row, 1, checkDesc)

}

func (u *UI) createCheckPage(config checks.Config) {
	u.primaryCheck = tview.NewTable()
	u.primaryCheck.SetBorder(true)
	u.primaryCheck.SetTitle("  Release image URL  ")
	u.primaryCheck.SetBorderColor(newt.ColorBlack)
	u.primaryCheck.SetBackgroundColor(newt.ColorGray)
	u.primaryCheck.SetTitleColor(newt.ColorBlack)
	u.setCheckWidget(u.primaryCheck, 1, checks.CheckTypeReleaseImagePull, config.ReleaseImageURL, config)

	u.checks = tview.NewTable()
	u.checks.SetBorder(true)
	u.checks.SetTitle("  Additional checks  ")
	u.checks.SetBorderColor(newt.ColorBlack)
	u.checks.SetBackgroundColor(newt.ColorGray)
	u.checks.SetTitleColor(newt.ColorBlack)
	u.setCheckWidget(u.checks, 0, checks.CheckTypeReleaseImageHostDNS, "nslookup %s", config)
	u.setCheckWidget(u.checks, 1, checks.CheckTypeReleaseImageHostPing, "ping %s", config)
	u.setCheckWidget(u.checks, 2, checks.CheckTypeReleaseImageHttp, "%s responds to http GET", config)

	u.details = tview.NewTextView()
	u.details.SetBorder(true)
	u.details.SetTitle("  Check Errors  ")
	u.details.SetDynamicColors(true)
	u.details.SetBorderColor(newt.ColorBlack)
	u.details.SetBackgroundColor(newt.ColorGray)
	u.details.SetTitleColor(newt.ColorBlack)

	u.form = tview.NewForm()
	u.form.SetBorder(false)
	u.form.SetBackgroundColor(newt.ColorGray)
	u.form.SetButtonsAlign(tview.AlignCenter)
	u.form.AddButton(CONFIGURE_BUTTON, func() {
		u.ShowNMTUI(nil)
	})
	u.form.AddButton(QUIT_BUTTON, func() {
		u.app.Stop()
	})
	u.form.SetButtonActivatedStyle(tcell.StyleDefault.Background(newt.ColorRed).
		Foreground(newt.ColorGray))
	u.form.SetButtonStyle(tcell.StyleDefault.Background(newt.ColorGray).
		Foreground(newt.ColorBlack))
	u.form.SetInputCapture(func(event *tcell.EventKey) (eventKey *tcell.EventKey) {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyRight:
			if _, index := u.form.GetFocusedItemIndex(); index == (u.form.GetButtonCount() - 1) {
				u.app.SetFocus(u.form.GetButton(0))
				eventKey = nil
			} else {
				u.app.SetFocus(u.form.GetButton(index + 1))
				eventKey = event
			}
		case tcell.KeyBacktab, tcell.KeyLeft:
			if _, index := u.form.GetFocusedItemIndex(); index == 0 {
				u.app.SetFocus(u.form.GetButton(1))
				eventKey = nil
			} else {
				u.app.SetFocus(u.form.GetButton((index) - 1))
				eventKey = event
			}
		case tcell.KeyEsc:
			u.app.Stop()
		default:
			eventKey = event
		}
		return
	})

	// SetRows(number-of-lines-to-give-1st-row,
	//         number-of-lines-to-give-2nd-row,..etc)
	// 0 means fill
	u.grid = tview.NewGrid().SetRows(5, 6, 0, 3).SetColumns(0).
		AddItem(u.primaryCheck, 0, 0, 1, 1, 0, 0, false).
		AddItem(u.checks, 1, 0, 1, 1, 0, 0, false).
		AddItem(u.details, 2, 0, 1, 1, 0, 0, false).
		AddItem(u.form, 3, 0, 1, 1, 0, 0, false)
	u.grid.SetTitle("  Agent installer network boot setup  ")
	u.grid.SetTitleColor(newt.ColorRed)
	u.grid.SetBorder(true)
	u.grid.SetBackgroundColor(newt.ColorGray)
	u.grid.SetBorderColor(tcell.ColorBlack)

	width := 80
	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(u.grid, 0, 8, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)

	u.pages.SetBackgroundColor(newt.ColorBlue)

	u.pages.AddPage(PAGE_CHECKSCREEN, flex, true, true)

	u.app.SetRoot(u.pages, true).SetFocus(u.form)
}
