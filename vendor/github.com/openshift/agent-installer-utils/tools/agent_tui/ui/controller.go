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

func (c *Controller) receivedPrimaryCheck(numChecks int) bool {
	_, found := c.checks[checks.CheckTypeReleaseImagePull]
	return found
}

func (c *Controller) Init(numChecks int) {

	c.ui.ShowSplashScreen()

	go func() {
		for {
			res := <-c.channel
			c.updateState(res)

			// Warming up, wait for at least
			// for the primary check
			if !c.receivedPrimaryCheck(numChecks) {
				continue
			}

			// When nmtui is shown the UI is suspended, so
			// let's skip any update
			if c.ui.IsNMTuiActive() {
				continue
			}

			// Pull check is always updated
			if res.Type == checks.CheckTypeReleaseImagePull {
				c.ui.SetPullCheck(res)
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

			c.updateCheckWidgets(res)

			// A check failed while waiting for the countdown. Timeout dialog must be stopped
			if !c.state && c.ui.IsTimeoutDialogActive() {
				c.ui.app.QueueUpdate(func() {
					c.ui.cancelUserPrompt()
				})
			}

			// A previously failed checked passed, so if the user never interacted with the ui
			// then it's safe to display the timeout dialog
			if c.state && !c.ui.IsTimeoutDialogActive() && !c.ui.IsDirty() {
				c.ui.app.QueueUpdateDraw(func() {
					c.ui.ShowTimeoutDialog()
				})
			}

		}
	}()
}

func (c *Controller) updateCheckWidgets(res checks.CheckResult) {

	// If everything is fine, clean additional checks
	// and details section, and skip the update
	if c.state {
		c.ui.app.QueueUpdateDraw(func() {
			c.ui.HideAdditionalChecks()
		})
		return
	} else {
		c.ui.app.QueueUpdateDraw(func() {
			c.ui.ShowAdditionalChecks()
		})
	}

	// Update the additional check widgets
	switch res.Type {
	case checks.CheckTypeReleaseImageHostDNS:
		c.ui.SetDNSCheck(res)
	case checks.CheckTypeReleaseImageHostPing:
		c.ui.SetPingCheck(res)
	case checks.CheckTypeReleaseImageHttp:
		c.ui.SetHttpGetCheck(res)
	}
}
