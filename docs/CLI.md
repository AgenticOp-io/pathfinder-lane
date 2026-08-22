# Command-line reference

## pathfinder (main GUI)

| Flag | Purpose |
| --- | --- |
| `-install-gui` | Graphical install / relocate binary |
| `-install` | CLI install to AppData |
| `-setup solo\|o365\|google` | Set access mode |
| `-logo <path>` | Custom window icon |
| `-vault-unlock` | Headless vault unlock (automation) |

Environment:

| Variable | Purpose |
| --- | --- |
| `PATHFINDERSSH_LOGO` | Custom logo path |

## pfinstall

Standalone install wizard binary from `cmd/pfinstall`.

## Install.ps1 (packaging)

| Parameter | Purpose |
| --- | --- |
| `-install-gui` | Show Fyne wizard |
| `-setup solo\|o365\|google` | Silent access mode |

## Headless tools

| Command | Purpose |
| --- | --- |
| `crawl` | BFS network discovery |
| `capture` | Config capture runner |
| `mapview` | Local map HTTP viewer |
| `pfvault` | Vault CLI utilities |

Run from repo root after `go build ./cmd/<name>`.
