# Command-line reference

## pfinstall (primary installer)

| Flag | Purpose |
| --- | --- |
| `-install` | Copy bundle to `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\`, create shortcuts |
| `-update` | Refresh installed binaries |
| `-install-gui` | Graphical install wizard |
| `-uninstall` | Remove AppData install and shortcuts |
| `-setup solo\|o365\|google` | Set access mode during install |
| `-enroll` | Complete cloud OAuth during CLI install |
| `-from <dir>` | Bundle folder (default: beside `pfinstall.exe`) |
| `-version` | Print installer version |

Examples:

```cmd
pfinstall.exe -install -setup solo
pfinstall.exe -from dist\windows -update
pfinstall.exe -uninstall
```

## pathfinder (main GUI)

Secondary install flags (same behavior; prefer `pfinstall.exe` for distribution):

| Flag | Purpose |
| --- | --- |
| `-install-gui` | Graphical install / relocate binary |
| `-install` | CLI install to AppData |
| `-uninstall` | Remove AppData install |
| `-setup solo\|o365\|google` | Set access mode (with `-install`) |
| `-logo <path>` | Custom window icon |
| `-vault-unlock` | Headless vault unlock (automation) |

Environment:

| Variable | Purpose |
| --- | --- |
| `PATHFINDERSSH_LOGO` | Custom logo path |

## Headless tools

| Command | Purpose |
| --- | --- |
| `pfseed` | Headless seeds, Auvik sync (`pfseed sync-auvik`) |
| `crawl` | BFS network discovery |
| `capture` | Config capture runner |
| `mapview` | Local map HTTP viewer |
| `pfvault` | Vault CLI utilities |

Run from repo root after `go build ./cmd/<name>` or from the installed AppData `bin` folder.
