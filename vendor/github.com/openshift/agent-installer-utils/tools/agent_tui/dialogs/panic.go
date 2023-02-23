package dialogs

import (
	"github.com/rivo/tview"
)

func PanicDialog(app *tview.Application, err error) {
	panicDialog := tview.NewModal().
		SetText(err.Error()).
		AddButtons([]string{"Quit"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.Stop()
		})
	app.SetRoot(panicDialog, false)
}
