// Script picker / runner dialogs.
package ui

import (
	"context"
	"fmt"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/scripts"
)

// ScriptSender is what the host supplies so scripts never import the shell.
type ScriptSender interface {
	SendActive(text string)
	SendAll(text string)
}

// ShowRunScriptDialog lets the operator pick a script and run it.
// cancel holds a running cancel func so a second run can stop the first.
func ShowRunScriptDialog(w fyne.Window, file scripts.File, sender ScriptSender, running *atomic.Pointer[context.CancelFunc]) {
	if w == nil || sender == nil {
		return
	}
	if len(file.Scripts) == 0 {
		dialog.ShowInformation("Scripts", "No scripts defined. Edit scripts.yaml first.", w)
		return
	}

	names := make([]string, len(file.Scripts))
	for i, sc := range file.Scripts {
		scope := sc.Scope
		if scope == "" {
			scope = "active"
		}
		names[i] = fmt.Sprintf("%s  (%s, %d steps)", sc.Name, scope, len(sc.Steps))
	}
	sel := widget.NewSelect(names, nil)
	sel.SetSelectedIndex(0)
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord

	form := container.NewVBox(
		widget.NewLabel("Choose a script to send into the terminal:"),
		sel,
		status,
	)

	d := dialog.NewCustomConfirm("Run Script", "Run", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		idx := sel.SelectedIndex()
		if idx < 0 || idx >= len(file.Scripts) {
			return
		}
		sc := file.Scripts[idx]
		if running != nil {
			if prev := running.Load(); prev != nil {
				(*prev)()
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		if running != nil {
			running.Store(&cancel)
		}
		status.SetText("Running " + sc.Name + "…")
		go func() {
			err := scripts.Run(ctx, sc, sender)
			fyne.Do(func() {
				if running != nil {
					running.Store(nil)
				}
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				dialog.ShowInformation("Scripts", "Finished: "+sc.Name, w)
			})
		}()
	}, w)
	d.Resize(fyne.NewSize(480, 220))
	d.Show()
}
