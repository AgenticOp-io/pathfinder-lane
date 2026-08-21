# Contributing to PathfinderSSH

This tree is a **fork** of Scott Peterman’s upstream, checked out under the
AgenticOps engines layout so we can develop locally and open PRs upstream.

| Remote | URL | Use |
| --- | --- | --- |
| `upstream` | https://github.com/scottpeterman/pathfinderssh | Fetch / rebase; PRs target this |
| `origin` | https://github.com/AgenticOp-io/pathfinderssh | Push feature branches |

Packaging, Windows installers, and prebuilt binaries live next door:

`C:\Users\david\AgenticOps\products\pathfinder`

A junction also exists at `C:\Users\david\projects\pathfinderssh` → this engine.

## One-time setup

```powershell
cd C:\Users\david\AgenticOps\engines\pathfinderssh
git fetch upstream
git checkout main
git merge --ff-only upstream/main
```

**Build deps (GUI):** Go 1.22+ and a C compiler on PATH (Fyne needs CGO).
On Windows: MSYS2 `mingw-w64-x86_64-gcc`, or TDM-GCC / WinLibs. Then:

```powershell
$env:CGO_ENABLED = "1"
.\build-windows.ps1 -Targets pathfinder
```

CLI-only tools (no Fyne) can build with `CGO_ENABLED=0`, e.g. `pfseed`:

```powershell
$env:CGO_ENABLED = "0"
go test ./cmd/pfseed
go build -o ..\..\products\pathfinder\windows\pfseed.exe ./cmd/pfseed
```

## Branch workflow (PR to upstream)

```powershell
git fetch upstream
git checkout -b feature/short-name upstream/main
# ... commit ...
git push -u origin HEAD
gh pr create --repo scottpeterman/pathfinderssh --base main --head AgenticOp-io:feature/short-name
```

Keep `main` aligned with upstream; do not develop long-lived work on `main`.

## What belongs where

| Change | Land in |
| --- | --- |
| App / Fyne / crawl / vault / sessions | this engine → PR upstream |
| `START.bat`, seed wizard, SecureCRT import packaging | `products\pathfinder` |
| Frontend CRT-replacement roadmap | `products\pathfinder\ROADMAP-FRONTEND.md` |

Upstream is GPL-3.0. Match existing code style and security non-negotiables
(credentials, host keys, read-only crawl) documented in `SECURITY.md` / `ROADMAP.md`.
