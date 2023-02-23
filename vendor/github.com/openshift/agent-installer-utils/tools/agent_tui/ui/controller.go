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
	c.state = true

	for _, res := range c.checks {
		if !res.Success {
			c.state = false
			break
		}
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
		c.ui.app.QueueUpdateDraw(func() {
			if res.Success {
				c.ui.markCheckSuccess(0, 0)
			} else {
				c.ui.markCheckFail(0, 0)
				c.ui.appendNewErrorToDetails("Release image pull error", res.Details)
			}
		})
	case checks.CheckTypeReleaseImageHostDNS:
		c.ui.app.QueueUpdateDraw(func() {
			if res.Success {
				c.ui.markCheckSuccess(1, 0)
			} else {
				c.ui.markCheckFail(1, 0)
				c.ui.appendNewErrorToDetails("nslookup failure", res.Details)
			}
		})
	case checks.CheckTypeReleaseImageHostPing:
		c.ui.app.QueueUpdateDraw(func() {
			if res.Success {
				c.ui.markCheckSuccess(2, 0)
			} else {
				c.ui.markCheckFail(2, 0)
				c.ui.appendNewErrorToDetails("ping failure", res.Details)
			}
		})
	case checks.CheckTypeReleaseImageHttp:
		c.ui.app.QueueUpdateDraw(func() {
			if res.Success {
				c.ui.markCheckSuccess(3, 0)
			} else {
				c.ui.markCheckFail(3, 0)
				c.ui.appendNewErrorToDetails("http server not responding", res.Details)
			}
		})
	}
}
