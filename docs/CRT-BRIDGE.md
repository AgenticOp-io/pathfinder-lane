# Pathfinder CRT Bridge (standalone SecureCRT companion)

This is a **separate installer** from Pathfinder. Pathfinder does not rewrite SecureCRT sessions. Upstream PathfinderSSH is not ours; this companion ships on the MSP fork as `pfcrt-install` / `pathfinder-crt`.

It launches official **AuvikTunnel.exe**, **FortiClient**, **WireGuard**, and **Zscaler Client Connector (ZSACli)** tools only. It does not embed those protocols.

## Automation modes

Pick one in `pfcrt-install`:

| Mode | Auvik | Customer VPN | Who gets a localhost proxy |
| --- | --- | --- | --- |
| **mixed** | Only from `Folder=AuvikTenant` | Only from the folder → VPN map | Mapped folders only |
| **forticlient** (VPN only) | No | FortiClient, WireGuard, Zscaler ZPA | Mapped CRT folders |
| **auvik** | Only from the folder map | No | Mapped Auvik folders only |

Setup is the map. CRT folder names, FortiClient connection names, WireGuard interface names, Zscaler ZPA, and Auvik domains **do not have to match**. After you bind “what they have” to “what they need” once, opening a session connects the right path automatically.

Map values:

| They have | Folder= value |
| --- | --- |
| FortiClient connection | `marshall-ssl` or `forticlient:marshall-ssl` |
| WireGuard tunnel | `wireguard:acme` |
| Zscaler Private Access | `zscaler:zpa` |
| Zscaler partner tenant | `zscaler:zpa:user@partner` |

The agent is exclusive across full-tunnel customer VPNs: connecting one FortiClient / WireGuard / ZPA tears the others down first. It never disables Zscaler Internet Access (ZIA). Passwords are never passed on the command line. Zscaler CLI must be enabled in the Client Connector app profile (4.4+). WireGuard tunnels should already exist in the official WireGuard app (`WireGuardTunnel$` services). MFA still appears in the vendor UI.

Leave the default FortiClient tunnel blank unless unmapped folders should share one connection. Missing CLIs are logged; the session still connects to the original host if that path is reachable.

## What install does

1. Copies `pathfinder-crt.exe`, `pfcrt-install.exe`, `pflane.exe`, and `AuvikTunnel.exe` to `%LOCALAPPDATA%\PathfinderCRT-Bridge\bin\`.
2. Stores config in `%USERPROFILE%\.pathfinder-crt\` (independent of Pathfinder’s `~/.pathfinderssh`).
3. Backs up the SecureCRT customer folder under `~/.pathfinder-crt/crt-backup/<stamp>/`.
4. Rewrites matching SSH sessions to `127.0.0.1:<front-port>` (ports 52000–61999).
5. Leaves unmatched sessions on standard SSH (original host and port).
6. Starts the agent at logon **when sessions were rewritten onto localhost**. Every five minutes it re-checks and updates new or removed customers. Linux/Mac: systemd user unit or launchd agent (`pflane serve`) — not for OpenSSH-only laptops.

On first standalone install, existing `~/.pathfinderssh/crt-bridge.json` is copied into `~/.pathfinder-crt` so already-rewritten sessions keep their originals.

## Installer (`pfcrt-install.exe`)

Double-click **`pfcrt-install.exe`** (or `.\build-windows.ps1 -Targets crt` then run `dist\windows\pfcrt-install.exe`).

| Action | What happens |
| --- | --- |
| **Install** (first time) | Backup the SecureCRT customer folder, rewrite matching SSH sessions to localhost, start the agent at logon |
| **Update** (already installed) | Stop the old agent, refresh binaries, rewrite SecureCRT sessions that need Auvik and/or FortiClient, restart the agent |
| **Uninstall** | Restore original SSH host/port in the `.ini` files and remove AppData binaries |

If SecureCRT is open, session files still update on disk. Open a **new** session (or restart SecureCRT) so it reads the new host/port.

```
pfcrt-install.exe                         graphical wizard (install or update)
pfcrt-install.exe -install -mode mixed
pfcrt-install.exe -update
pfcrt-install.exe -uninstall

pathfinder-crt              # background agent (after install)
pathfinder-crt -sync
pathfinder-crt -uninstall
```

Build: `.\build-windows.ps1 -Targets crt` (also builds `pflane.exe`). Linux/Mac: `go build -o pflane ./cmd/pflane`. Cross-platform CLI (OpenSSH, CRT, PuTTY): [PFLANE.md](PFLANE.md).

Uninstall restores original host/port in the `.ini` files and removes the Startup shortcut. The backup folder is kept.
