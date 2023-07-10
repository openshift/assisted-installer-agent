package agent_tui

import (
	"fmt"
	"log"

	"github.com/openshift/agent-installer-utils/tools/agent_tui/checks"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/ui"
	"github.com/rivo/tview"
)

func App(app *tview.Application, config checks.Config, checkFuncs ...checks.CheckFunctions) {

	if err := prepareConfig(&config); err != nil {
		log.Fatal(err)
	}

	var appUI *ui.UI
	if app == nil {
		app = tview.NewApplication()
	}
	appUI = ui.NewUI(app, config)
	controller := ui.NewController(appUI)
	engine := checks.NewEngine(controller.GetChan(), config, checkFuncs...)

	controller.Init(engine.Size())
	engine.Init()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func prepareConfig(config *checks.Config) error {
	// Set hostname
	hostname, err := checks.ParseHostnameFromURL(config.ReleaseImageURL)
	if err != nil {
		return err
	}
	config.ReleaseImageHostname = hostname

	// Set scheme
	schemeHostnamePort, err := checks.ParseSchemeHostnamePortFromURL(config.ReleaseImageURL, "https://")
	if err != nil {
		return fmt.Errorf("error creating <scheme>://<hostname>:<port> from releaseImageURL: %s", config.ReleaseImageURL)
	}
	config.ReleaseImageSchemeHostnamePort = schemeHostnamePort

	return nil
}
