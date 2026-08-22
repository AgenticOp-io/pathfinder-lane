# Cursor AI integration

Optional troubleshooting addon using Cursor **Cloud Agents API**.

![Cursor pane](images/msp-cursor-pane.svg)

## Enable

1. Settings → **Cursor** — enter API key / account
2. Settings → **Ops** — enable Troubleshoot addon
3. Ops menu or toolbar → **Troubleshoot** opens modal; side pane available when session is active

## What it does

- Sends session context (hostname, recent terminal output snippets) to Cursor cloud
- Model picker from `ListModels` API
- Does not replace on-device vault or SSH — advisory layer only

## Code

- `internal/cursorapi/` — HTTP client
- `internal/ui/cursorpane.go`, `troubleshootmodal.go`

## Security

- API key stored locally in settings
- Review Cursor data handling before enabling in regulated environments
- Disable Ops addon for air-gapped deployments
