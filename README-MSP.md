# PathfinderSSH MSP

MSP edition of PathfinderSSH: built-in **Customers** / **Unassigned** inventory,
customer-scoped crawl wizard, drag-and-drop folders, vault password save.

This repository is the product fork. Portable terminal/SSH fixes are contributed
upstream when appropriate.

## Build

```bash
go build -ldflags "-s -w -H windowsgui" -o pathfinder.exe ./cmd/pathfinder
```

Windows GUI subsystem (`-H windowsgui`) is required so a blank console window
does not appear beside the app.

## Layout

- `Customers/` — one folder per customer (create in the connection pane)
- `Unassigned/` — flat list for connections not filed under a customer

Legacy SecureCRT `3_Customers` trees are migrated into this layout on startup.

## Logo

Default window / About / `.exe` icon is the **AgenticOps** mark.

Override without rebuilding:

- `%USERPROFILE%\.pathfinderssh\logo.png`
- `%LOCALAPPDATA%\PathfinderSSH-MSP\logo.png`
- env `PATHFINDERSSH_LOGO`, or `-logo path\to\file.png`

Rebuild with a new PE icon via `cmd/pathfinder/winres/` + `go-winres` (see `internal/ui/assets/README.md`).
