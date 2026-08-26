# lane — keep the SSH client you already use

Engineers stay in OpenSSH, PuTTY, SecureCRT, or Pathfinder. Setup is a **customer map** (names never have to match). After that, opening a session brings the VPN up.

| They open | What we write | Daemon? |
| --- | --- | --- |
| `ssh` (Windows / Linux / Mac) | `~/.ssh/config` Include + ProxyCommand | No |
| **PuTTY** | Local proxy command on the saved session | No |
| **Pathfinder** | Nothing — Connect calls the same VPN ensure | No |
| **SecureCRT** | Mapped sessions → `127.0.0.1` | Yes — `lane serve` (Windows logon: `lane-crt`) |

Unmapped sessions stay on standard SSH.

## Setup (once per laptop)

```
lane setup
```

That walk:

1. Lists FortiClient / WireGuard / Zscaler tunnels on this PC.
2. Asks the **customer name you already use** (CRT folder, Pathfinder `Customers/Acme`, whatever you call them).
3. Lists saved sessions that are not under a customer yet (PuTTY `fw-01`, OpenSSH `Host edge`) and asks which customer they belong to.
4. Writes OpenSSH, PuTTY, and SecureCRT.

Same thing without the prompts:

```
lane map-set Acme wireguard:acme
lane bind fw-01 Acme
lane bind edge Acme
lane create-all
```

Then they work as they already do: `ssh edge`, the PuTTY session named `fw-01`, Pathfinder tree, or CRT.

## What “bind” is for

CRT and Pathfinder already have folders. PuTTY and OpenSSH often do not. `lane bind NAME Acme` is the human map for those tools — it does not rename the session.

## Install

- Windows CRT wizard (`lane-install`) also writes OpenSSH/PuTTY after the folder map. Binaries land in `%LOCALAPPDATA%\Lane\bin\`.
- Linux/Mac: `go build -o lane ./cmd/lane && ./lane setup` (copies itself to `~/.lane-app/bin/lane`).

`create-all` installs a **stable** `lane` path so OpenSSH ProxyCommand does not point at a `go run` cache, and puts that binary on the user PATH (`%LOCALAPPDATA%\Lane\bin` on Windows, `~/.local/bin` on Linux/Mac; Mac also appends a `# agenticop-lane` line to `~/.zprofile` when needed). Open a new shell after setup.

SecureCRT login autostart (`lane serve`, or `lane-crt` when that binary is present) is registered **only when CRT sessions were rewritten onto localhost**. OpenSSH/PuTTY proxy mode does not start a daemon. Toggle with `lane autostart` / `lane autostart off`.

Mapped OpenSSH hosts that already have a Pathfinder jump hop (or `ProxyJump` in `~/.ssh/config` / `hosts.json`) get a generated jump `Host` plus `ProxyJump` — VPN Ensure runs on the bastion hop. Pathfinder Connect already Ensures, then uses `JumpSpec`.

The splice prefers the customer VPN adapter when we can name it (WireGuard iface, Forti SSL adapter, Zscaler). Pathfinder Connect uses the same bind for its first TCP hop (direct or nearest bastion). Unmatched adapters fail-open to the default route. Overlapping office `10.x` vs customer `10.x` still needs that bind to succeed.

## Commands

| Command | Purpose |
| --- | --- |
| `lane setup` | Interactive map + bind + create-all |
| `lane bind NAME FOLDER` | This session belongs to that customer |
| `lane unbind NAME` | Remove a bind |
| `lane map-set FOLDER TARGET` | Customer → VPN |
| `lane map-auvik FOLDER TENANT` | Customer → Auvik |
| `lane create-all` | Write every client present |
| `lane status` / `list` / `list-vpns` / `map` | See what is mapped |
| `lane serve` | SecureCRT localhost agent |
| `lane ssh ALIAS` | OpenSSH via generated aliases |
| `lane ensure TARGET` | Bring one VPN up |
| `lane restore-putty` | Undo PuTTY rewrite |
| `lane autostart [off]` | CRT login agent (only needed when sessions were rewritten to localhost) |

Official vendor CLIs only. Passwords never on the command line. One customer VPN at a time (Forti / WireGuard / ZPA); ZIA is never disabled.
