# PathfinderSSH MSP

**AgenticOps fork** of [PathfinderSSH](https://github.com/scottpeterman/pathfinderssh) (Scott Peterman).  
Upstream PathfinderSSH is not an AgenticOps product; this MSP edition is.

| | |
|--|--|
| **Our repo** | https://github.com/AgenticOp-io/pathfinderssh-msp |
| **Upstream** | https://github.com/scottpeterman/pathfinderssh |

Built-in **Customers** / **Unassigned** inventory, customer-scoped crawl wizard, drag-and-drop folders, vault password save.

Portable terminal/SSH fixes may be contributed upstream when appropriate; do not present this tree as upstream PathfinderSSH.

## Build

```bash
go build -ldflags "-s -w -H windowsgui" -o pathfinder.exe ./cmd/pathfinder
```

Windows GUI subsystem (`-H windowsgui`) is required so a blank console window
does not appear beside the app.

## Layout

- `Customers/` — one folder per customer (create in the connection pane)
- `Unassigned/` — flat list for connections not filed under a customer

**Maps** are also per customer:

`%USERPROFILE%\.pathfinderssh\maps\<Customer>\crawl-YYYY-MM-DD.json`

Open via toolbar **Map** (pick customer → pick map). Import topology map files
sessions under `Customers/<customer>/<site>/`.

**New customer crawl seeds:** File → Import customer crawl seeds (CSV)…
downloads a template (`host,name,protocol,port,username,folder,notes`).

Legacy SecureCRT `3_Customers` trees are migrated into this layout on startup.

## Logo

Default window / About / `.exe` icon is the **AgenticOps** mark (MSP packaging).

Override without rebuilding:

- `%USERPROFILE%\.pathfinderssh\logo.png`
- `%LOCALAPPDATA%\PathfinderSSH-MSP\logo.png`
- env `PATHFINDERSSH_LOGO`, or `-logo path\to\file.png`

Rebuild with a new PE icon via `cmd/pathfinder/winres/` + `go-winres` (see `internal/ui/assets/README.md`).
