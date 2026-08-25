# Pathfinder Lane

**Last-mile laptop plane** — engineers keep SecureCRT, PuTTY, and OpenSSH. Setup is a customer map; opening a session brings the right VPN up.

| | |
|--|--|
| **Repo** | https://github.com/AgenticOp-io/pathfinder-lane |
| **Upstream engine** | [PathfinderSSH](https://github.com/scottpeterman/pathfinderssh) (Scott Peterman, GPL-3.0) |
| **License** | GPL-3.0 — see `LICENSE` and `NOTICE` |

This is a **GPL-3.0 derivative** of PathfinderSSH. AgenticOps does **not** own upstream PathfinderSSH. The previous MSP GitHub repo (`pathfinderssh-msp`) is retired; product work lives here.

CLI: [`docs/PFLANE.md`](docs/PFLANE.md) (`pflane setup` then the SSH client they already use).

![MSP main window ΓÇö session tree and toolbar](docs/images/msp-main-window.svg)

## Quick start

Build or download the Windows bundle, then run the installer:

```cmd
dist\windows\pfinstall.exe
```

Or silent install with solo mode (no cloud sign-in):

```cmd
dist\windows\pfinstall.exe -install -setup solo
```

Pick **Solo**, **Microsoft 365**, or **Google** in the graphical wizard when you double-click `pfinstall.exe`.
Full guide: [docs/INSTALL.md](docs/INSTALL.md)

## Documentation

| Start here | |
| --- | --- |
| [docs/README.md](docs/README.md) | Documentation index |
| [docs/MSP-SYNOPSIS.md](docs/MSP-SYNOPSIS.md) | What we built |
| [docs/MSP-FEATURES.md](docs/MSP-FEATURES.md) | Feature catalog + UI tour |
| [docs/MSP-INTEGRATION-STACK.md](docs/MSP-INTEGRATION-STACK.md) | Cohesive SSH loop - PSA + inventory + vault |
| [docs/INTEGRATIONS.md](docs/INTEGRATIONS.md) | Integration architecture |
| [SECURITY.md](SECURITY.md) | Security policy |

## MSP highlights

- **Customers / Unassigned** inventory tree with SecureCRT import
- **Ops chrome** - macro button bar, SFTP tabs, port forwards, tile layout
- **Cohesive integrations** (MSP cloud sign-in only) - PSA customers -> RMM IPs -> vault passwords -> SSH
  - Inventory: Auvik, Domotz, NinjaOne, Datto RMM, Automate, N-central
  - Vault: IT Glue, Hudu, Passportal
  - PSA: ConnectWise, Autotask, Halo (+ JSON file adapter)
- **Sign-in** - Solo hides integration UI; Microsoft 365 or Google unlocks the MSP stack
- **pflane** - keep SecureCRT, PuTTY, or OpenSSH (`create-all` on Windows/Linux/Mac)
- **Cursor AI** (optional) - troubleshoot pane + Cloud Agents API

## Build

```bash
go build -ldflags "-s -w -H windowsgui" -o pathfinder.exe ./cmd/pathfinder
```

Windows requires `-H windowsgui` so no console window appears beside the app.

## Data locations

| Path | Contents |
| --- | --- |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\` | Installed binary, enrollment |
| `%USERPROFILE%\.pathfinderssh\` | sessions.yaml, vault, maps, scripts, settings |

## Contributing

AgenticOp-io maintains the MSP fork. Portable fixes may be offered upstream via PR to `scottpeterman/pathfinderssh`. See [docs/UPSTREAM.md](docs/UPSTREAM.md) and [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).
