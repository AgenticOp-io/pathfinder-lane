# Contributing to Pathfinder Lane

| Remote | URL | Use |
| --- | --- | --- |
| `origin` | https://github.com/AgenticOp-io/pathfinder-lane | Push feature branches (`candidate/*`) |
| `upstream` | https://github.com/scottpeterman/pathfinderssh | Fetch portable engine fixes |

Packaging and Windows installers: `products/pathfinder-msp/` in the AgenticOps umbrella.

## Build

**GUI (Fyne):** Go 1.22+, C compiler on PATH (`CGO_ENABLED=1`).

```powershell
$env:CGO_ENABLED = "1"
go build -ldflags "-s -w -H windowsgui" -o pathfinder.exe ./cmd/pathfinder
```

Windows requires `-H windowsgui` for the main binary.

CLI-only tools can build with `CGO_ENABLED=0`.

## Branch workflow

```powershell
git fetch origin
git checkout -b candidate/short-name
# ... commit ...
git push -u origin HEAD
```

Agents push to `candidate/*` only — merge to `main` is human / PR.

## What belongs where

| Change | Land in |
| --- | --- |
| Last-mile plane, pflane, CRT bridge | this repo (`pathfinder-lane`) |
| Portable crawl / vault / terminal fixes | PR to `scottpeterman/pathfinderssh` |
| Installers / `pfinstall.exe` | `cmd/pfinstall`, `dist/windows/` |

Match security non-negotiables in [SECURITY.md](../SECURITY.md). UI roadmap: [ROADMAP-FRONTEND.md](../ROADMAP-FRONTEND.md).
