# PathfinderSSH MSP — synopsis

**What it is:** A single Windows desktop app for network engineers — SSH/telnet/serial terminals, discovery crawl, config capture, topology maps, and an MSP-oriented session inventory — extended for managed service providers with customer folders, RMM/doc integrations, and optional cloud sign-in.

**What it is not:** A hosted SaaS, a replacement for your PSA, or upstream PathfinderSSH itself. This tree is a **GPL-3.0 derivative** maintained by AgenticOps (`pathfinderssh-msp`). The original engine and vision are Scott Peterman’s [PathfinderSSH](https://github.com/scottpeterman/pathfinderssh).

---

## Lineage

| Layer | Author / owner | Role |
| --- | --- | --- |
| **PathfinderSSH core** | Scott Peterman | Terminal, crawl, capture, map, YAML session tree, encrypted vault, click-to-connect loop |
| **MSP edition** | AgenticOp-io | Customers/Unassigned layout, ops chrome, integrations, auth, packaging |

Installed binary: `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pathfinder.exe`  
User data: `%USERPROFILE%\.pathfinderssh\` (sessions, vault, maps, scripts, settings)

---

## What ships today (MSP fork)

### Core loop (upstream + MSP polish)

- **Session tree** — nested folders, YAML on disk, filter, drag-and-drop
- **SSH / telnet / serial** — tabs, detach, scrollback, themes (including CRT green/amber)
- **Crawl** — BFS discovery, platform fingerprint, MikroTik neighbors, hypervisor guests
- **Capture** — content-addressed config store
- **Map** — browser viewer on loopback; click node → session dialog
- **Vault** — AES-256-GCM credential file; sessions reference vault by name

### MSP inventory model

- **`Customers/<client>/…`** — one folder per customer; nested sites
- **`Unassigned/…`** — everything else (legacy CRT imports)
- SecureCRT import wizard maps `3_Customers` → `Customers/`
- PSA customer list import (`psa-customers.json` adapter)
- Customer crawl seeds via CSV

### Ops chrome (SecureCRT-style)

- Top toolbar: Connect, Crawl, Capture, Map, Search, Scripts, Tabs, Settings
- Bottom **button bar** — YAML macros (`send:` / `script:`), optional **All tabs**
- **Tile / untile** terminals; Ctrl+Tab between tiled panes
- **SFTP** in tabs; **port forwards** (local/remote/SOCKS5)
- Session menu: capture, scrollback save, evidence pack, script recorder
- Read-only mode + change window (`internal/policy`)

### Install & sign-in

- **GUI installer** — `Install.ps1` → wizard: Solo / Microsoft 365 / Google
- **Solo** — no cloud OAuth; local enrollment only
- **Microsoft 365 / Google** — OIDC + PKCE in system browser; enrollment file + per-user session
- Packages: `internal/idp`, `internal/mspenroll`, `internal/mspauth`
- Tools: `pathfinder -install-gui`, `cmd/pfinstall`, `cmd/pfenroll`

### RMM / documentation integrations

| System | What Pathfinder does | What it does *not* do |
| --- | --- | --- |
| **Auvik** | List tenants/devices; sync IPs into `Customers/<client>/Auvik/`; merge by device id/IP/name; periodic sync; **AuvikTunnel** fallback on failed SSH | Export device passwords; headless SSH via API |
| **IT Glue** | List orgs/passwords; import plaintext into **vault** (API key + Password Access); link credentials to sessions under customer | Rotation, JIT bastion, compliance audits (planned extensions) |

Recommended flow: **Auvik** for targets → **IT Glue** for credentials → connect via vault.

### AI troubleshooting (optional addon)

- Settings → enable **Troubleshoot addon**
- **Cursor AI** right pane: gather scrollback, ask Cursor Cloud Agent, send commands to active SSH
- Troubleshoot modal (full evidence pack + optional repo for agent)
- `internal/cursorapi` — Cloud Agents API; key via `CURSOR_API_KEY` or settings

### Security posture

See [SECURITY.md](../SECURITY.md): vault-only secrets, host-key fail-closed, read-only crawl/capture allowlists, loopback map with token, no passwords in session YAML. Integration secrets live in settings JSON or env vars — never logged.

---

## Repository map (high level)

```
cmd/pathfinder/     Main GUI application
cmd/pfinstall/      GUI installer only
cmd/pfenroll/       Enrollment wizard only
cmd/pfvault/        Vault CLI
internal/auvik/     Auvik API client + sync
internal/itglue/    IT Glue API + vault/session link
internal/idp/       OIDC (Entra, Google)
internal/mspauth/   Auth orchestration
internal/mspenroll/ Org enrollment file
internal/cursorapi/ Cursor Cloud Agents
internal/ui/        Fyne UI (shell, settings, integrations)
products/pathfinder-msp/   Windows install scripts + binaries
docs/               This documentation set
```

---

## Maturity snapshot

| Area | Status |
| --- | --- |
| Terminal + inventory + crawl/map/capture | **Production-shaped** (upstream core) |
| MSP Customers layout + CRT import | **Shipped** |
| Ops chrome (bar, SFTP, forwards, tile) | **Shipped** |
| Install wizard + Solo/O365/Google | **Shipped** |
| Auvik sync + tunnel | **Shipped** |
| IT Glue vault import + session link | **Shipped** |
| Cursor AI pane + troubleshoot addon | **Shipped** (addon gated) |
| Live ConnectWise/Autotask APIs | **Not shipped** (JSON file adapter only) |
| JIT bastion, compliance cop, password rotation | **Roadmap** (see INTEGRATIONS.md) |

---

## Next reads

- [MSP-FEATURES.md](MSP-FEATURES.md) — where each feature lives in the UI
- [INTEGRATIONS.md](INTEGRATIONS.md) — middleware architecture
- [INSTALL.md](INSTALL.md) — get a build on your machine
