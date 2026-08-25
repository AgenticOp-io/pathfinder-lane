# Lane

**Last-mile laptop plane** — engineers keep SecureCRT, PuTTY, and OpenSSH. Setup is a customer map; opening a session brings the right VPN up.

| | |
|--|--|
| **Repo** | https://github.com/AgenticOp-io/lane |
| **CLI** | `lane` |
| **CRT companion** | `lane-install` / `lane-crt` (or `lane serve`) |
| **Upstream engine** | [PathfinderSSH](https://github.com/scottpeterman/pathfinderssh) (Scott Peterman, GPL-3.0) |
| **License** | GPL-3.0 — see `LICENSE` and `NOTICE` |

This is a **GPL-3.0 derivative** of PathfinderSSH. AgenticOps does **not** own upstream PathfinderSSH.

[PathfinderSSH MSP](https://github.com/AgenticOp-io/pathfinderssh-msp) is a **separate** product (the original Pathfinder concept). Last-mile work lives here.

## Quick start

```
lane setup
```

Then open the same session as today: `ssh`, PuTTY, SecureCRT, or Pathfinder. Mapped OpenSSH and PuTTY sessions do not need a daemon. SecureCRT rewritten onto localhost needs `lane serve` (Windows: `lane-crt` at logon).

Windows CRT wizard: `lane-install.exe`. Full CLI: [`docs/LANE.md`](docs/LANE.md). SecureCRT companion: [`docs/CRT-BRIDGE.md`](docs/CRT-BRIDGE.md).

```powershell
.\build-windows.ps1 -Targets crt
```

Linux/Mac:

```bash
go build -o lane ./cmd/lane
./lane setup
```

Config: `~/.lane` (Windows install binaries: `%LOCALAPPDATA%\Lane\bin`).

## Documentation

| Start here | |
| --- | --- |
| [docs/LANE.md](docs/LANE.md) | Map, bind, OpenSSH / PuTTY / CRT |
| [docs/CRT-BRIDGE.md](docs/CRT-BRIDGE.md) | SecureCRT companion |
| [docs/README.md](docs/README.md) | Index (includes inherited PathfinderSSH GUI docs) |
| [SECURITY.md](SECURITY.md) | Security policy |

## Contributing

AgenticOp-io maintains Lane. Portable engine fixes may be offered upstream via PR to `scottpeterman/pathfinderssh`. See [docs/UPSTREAM.md](docs/UPSTREAM.md) and [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).
