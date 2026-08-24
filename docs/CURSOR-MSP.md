# Cursor IDE ↔ PathfinderSSH MSP

Connect **Cursor Agent tools** to a **running PathfinderSSH MSP** window so the agent can list SSH tabs, read scrollback, and optionally type commands.

This is separate from **Cloud Agents** (Cursor API key / Troubleshoot pane), which cannot drive the Fyne desktop UI.

Cursor still uses its own MCP config file (`mcp.json`); the Pathfinder side is branded **MSP**.

## Architecture

```
Cursor IDE  --stdio-->  pathfinder-msp.exe  --HTTP-->  PathfinderSSH MSP (127.0.0.1:19790)
                              ^                              ^
                              |                              |
                     reads ~/.pathfinderssh/          writes same file on start
                     msp-bridge.json
```

| Piece | Role |
|-------|------|
| PathfinderSSH MSP | Localhost HTTP bridge while the app is open |
| `pathfinder-msp.exe` | Stdio server Cursor launches |
| `msp-bridge.json` | URL + token discovery under `~/.pathfinderssh/` |

## Setup

1. Build and install so both binaries land in `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\`:

   ```powershell
   .\build-windows.ps1 -Targets installers -Install
   ```

2. Start PathfinderSSH MSP. By default the Cursor bridge is **on** (Settings → Tools → **Cursor IDE bridge (MSP)**). Leave **Allow send** off until you want Cursor to type into live sessions.

3. Add to `%USERPROFILE%\.cursor\mcp.json` (merge with any existing servers):

   ```json
   {
     "mcpServers": {
       "pathfinder-msp": {
         "command": "C:\\Users\\YOURUSER\\AppData\\Local\\PathfinderSSH-MSP\\bin\\pathfinder-msp.exe"
       }
     }
   }
   ```

4. Restart Cursor (or reload MCP servers). In Agent chat, tools like `pathfinder_list_sessions` and `pathfinder_read_scrollback` should appear.

## Tools

| Tool | Purpose |
|------|---------|
| `pathfinder_health` | Bridge reachable? |
| `pathfinder_list_sessions` | Open SSH tabs |
| `pathfinder_active_session` | Tab that receives keyboard / macros |
| `pathfinder_read_scrollback` | Terminal history (`session_id` optional) |
| `pathfinder_send_command` | Type into a session (`confirm: true` required; needs Allow send) |

## Safety

- Bind is **127.0.0.1 only**; requests need the bearer token from `msp-bridge.json`.
- **Allow send** is off by default; Pathfinder read-only / change-window policy still applies when send is enabled.

## Env overrides (optional)

| Variable | Meaning |
|----------|---------|
| `PATHFINDER_MSP_URL` | Skip state file; use this base URL |
| `PATHFINDER_MSP_TOKEN` | Bearer token when using URL override |
| `PATHFINDER_HOME` | Alternate directory for `msp-bridge.json` |

## Troubleshooting

- **Bridge not running**: open PathfinderSSH MSP; check Settings → Tools → Enable bridge; confirm `~/.pathfinderssh/msp-bridge.json` exists while the app is up.
- **Unauthorized**: token mismatch — restart PathfinderSSH MSP so the state file matches settings.
- **Send forbidden**: enable **Allow send** in Settings, and ensure read-only mode is off.
