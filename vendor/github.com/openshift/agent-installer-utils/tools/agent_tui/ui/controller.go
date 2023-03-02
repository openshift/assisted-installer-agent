package ui

import (
	"github.com/openshift/agent-installer-utils/tools/agent_tui/checks"
)

// Controller
type Controller struct {
	ui      *UI
	channel chan checks.CheckResult

	checks map[string]checks.CheckResult
	state  bool
}

func NewController(ui *UI) *Controller {
	return &Controller{
		channel: make(chan checks.CheckResult, 10),
		ui:      ui,
		checks:  make(map[string]checks.CheckResult),
	}
}

func (c *Controller) GetChan() chan checks.CheckResult {
	return c.channel
}

func (c *Controller) updateState(cr checks.CheckResult) {
	c.checks[cr.Type] = cr

	switch cr.Type {
	case checks.CheckTypeReleaseImagePull:
		c.state = cr.Success
	}
}

func (c *Controller) receivedAllCheckResults(numChecks int) bool {
	return len(c.checks) >= numChecks
}

func (c *Controller) Init(numChecks int) {

	c.ui.ShowSplashScreen()

	go func() {
		for {
			res := <-c.channel
			c.updateState(res)

			// When nmtui is shown the UI is suspended, so
			// let's skip any update
			if c.ui.IsNMTuiActive() {
				continue
			}

			// Keep the checks page continuously updated
			c.updateCheckWidgets(res)

			// Warming up, wait for at least the first
			// set of check results
			if !c.receivedAllCheckResults(numChecks) {
				continue
			}

			// After receiving the initial results, let's
			// show the timeout dialog if required
			if c.ui.IsSplashScreenActive() {
				c.ui.app.QueueUpdateDraw(func() {
					c.ui.HideSplashScreen()
					if c.state {
						c.ui.ShowTimeoutDialog()
					} else {
						c.ui.returnFocusToChecks()
					}
				})
				continue
			}

			// A check failed while waiting for the countdown. Timeout dialog must be stopped
			if !c.state && c.ui.IsTimeoutDialogActive() {
				c.ui.app.QueueUpdate(func() {
					c.ui.cancelUserPrompt()
				})
			}

		}
	}()
}

func (c *Controller) updateCheckWidgets(res checks.CheckResult) {
	// Update the widgets
	switch res.Type {
	case checks.CheckTypeReleaseImagePull:
		c.ui.SetPullCheck(res)
	case checks.CheckTypeReleaseImageHostDNS:
		c.ui.SetDNSCheck(res)
	case checks.CheckTypeReleaseImageHostPing:
		c.ui.SetPingCheck(res)
	case checks.CheckTypeReleaseImageHttp:
		c.ui.SetHttpGetCheck(res)
	}
}
