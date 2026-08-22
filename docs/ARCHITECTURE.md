# Architecture overview

## Binaries

| Binary | Role |
| --- | --- |
| `pathfinder` | Main MSP GUI (`cmd/pathfinder`) |
| `pfinstall` | Standalone install wizard (`cmd/pfinstall`) |
| `crawl`, `capture`, `mapview`, … | Headless / auxiliary tools |

See [CLI.md](CLI.md) for flags.

## Data on disk

```
%USERPROFILE%\.pathfinderssh\
  sessions.yaml      # tree: Customers/, Unassigned/
  vault.enc            # AES-256-GCM credentials
  settings.json        # integrations, UI, auth enrollment
  maps/<Customer>/     # crawl topology JSON
  scripts/             # button bar macros, Python scripts
```

## Key packages

| Package | Role |
| --- | --- |
| `internal/sessions` | YAML tree, `Node` (Auvik + IT Glue fields) |
| `internal/vault` | Encrypted credential store |
| `internal/ui` | Fyne chrome, shell, settings, wizards |
| `internal/auvik` | Auvik REST sync |
| `internal/itglue` | IT Glue vault import + link |
| `internal/idp`, `mspauth`, `mspenroll` | OIDC sign-in |
| `internal/cursorapi` | Cursor Cloud Agents |
| `internal/portfwd` | Port forwarding |
| `internal/evidence` | Ticket evidence packs |

## UI shell

- `appchrome.go` — top toolbar
- `shell.go` — terminal tabs, tile layout, right pane (Cursor)
- `buttonbar.go` — bottom macro bar

Upstream engine internals: [UPSTREAM.md](UPSTREAM.md)

## MSP packaging

`products/pathfinder-msp/Install.ps1` — copies binary to `%LOCALAPPDATA%\PathfinderSSH-MSP\`
