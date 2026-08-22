package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/cursorapi"
)

// ShowCursorAccountDialog verifies the API key and optionally launches a cloud agent.
func ShowCursorAccountDialog(w fyne.Window, apiKey string) {
	if w == nil {
		return
	}
	cli := cursorapi.New(apiKey)
	status := widget.NewLabel("Checking Cursor account…")
	status.Wrapping = fyne.TextWrapWord

	prompt := widget.NewMultiLineEntry()
	prompt.SetPlaceHolder("Cloud agent prompt (optional)")
	prompt.SetMinRowsVisible(4)

	repo := widget.NewEntry()
	repo.SetPlaceHolder("https://github.com/org/repo (optional)")
	ref := widget.NewEntry()
	ref.SetPlaceHolder("starting ref (main)")
	ref.SetText("main")
	name := widget.NewEntry()
	name.SetPlaceHolder("agent display name (optional)")
	modelSel := widget.NewSelect([]string{}, nil)
	modelSel.PlaceHolder = "model (optional)"

	refresh := widget.NewButton("Refresh account", nil)
	launch := widget.NewButton("Launch cloud agent", nil)
	launch.Disable()

	doMe := func() {
		status.SetText("Checking…")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			me, err := cli.Me(ctx)
			models, modErr := cli.ListModels(ctx)
			fyne.Do(func() {
				if err != nil {
					status.SetText("Account check failed: " + err.Error() +
						"\n\nSet CURSOR_API_KEY or Settings → Ops → Cursor API key.\nDashboard: https://cursor.com/dashboard/api")
					launch.Disable()
					return
				}
				who := strings.TrimSpace(me.UserEmail)
				if who == "" {
					who = "(email not returned)"
				}
				keyName := strings.TrimSpace(me.APIKeyName)
				if keyName == "" {
					keyName = "(unnamed key)"
				}
				msg := fmt.Sprintf("Authenticated.\nUser: %s\nAPI key: %s", who, keyName)
				if modErr != nil {
					msg += "\nModels: (could not list — " + modErr.Error() + ")"
				} else if len(models) > 0 {
					ids := make([]string, 0, len(models))
					for _, m := range models {
						if id := strings.TrimSpace(m.ID); id != "" {
							ids = append(ids, id)
						}
					}
					modelSel.Options = ids
					if len(ids) > 0 {
						modelSel.SetSelected(ids[0])
					}
					msg += fmt.Sprintf("\nModels: %d available", len(ids))
				}
				status.SetText(msg)
				launch.Enable()
			})
		}()
	}
	refresh.OnTapped = doMe
	launch.OnTapped = func() {
		text := strings.TrimSpace(prompt.Text)
		if text == "" {
			dialog.ShowInformation("Cursor", "Enter a prompt for the cloud agent.", w)
			return
		}
		req := cursorapi.CreateAgentRequest{
			Prompt: cursorapi.CreatePrompt{Text: text},
			Name:   strings.TrimSpace(name.Text),
		}
		if u := strings.TrimSpace(repo.Text); u != "" {
			req.Repos = []cursorapi.RepoSpec{{
				URL:         u,
				StartingRef: strings.TrimSpace(ref.Text),
			}}
		}
		if id := strings.TrimSpace(modelSel.Selected); id != "" {
			req.Model = &cursorapi.ModelRef{ID: id}
		}
		status.SetText("Launching agent…")
		launch.Disable()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			out, err := cli.CreateAgent(ctx, req)
			fyne.Do(func() {
				launch.Enable()
				if err != nil {
					status.SetText("Launch failed: " + err.Error())
					dialog.ShowError(err, w)
					return
				}
				msg := fmt.Sprintf("Agent %s (%s)\nRun %s (%s)",
					out.Agent.ID, out.Agent.Status, out.Run.ID, out.Run.Status)
				if out.Agent.URL != "" {
					msg += "\n" + out.Agent.URL
				}
				status.SetText(msg)
				dialog.ShowInformation("Cloud agent started", msg, w)
			})
		}()
	}

	body := container.NewBorder(
		container.NewVBox(status, container.NewHBox(refresh)),
		container.NewHBox(launch),
		nil, nil,
		container.NewVBox(
			widget.NewLabel("Launch (Cloud Agents API)"),
			name, modelSel, repo, ref, prompt,
		),
	)
	d := dialog.NewCustom("Cursor account API", "Close", body, w)
	d.Resize(fyne.NewSize(560, 480))
	d.Show()
	doMe()
}
