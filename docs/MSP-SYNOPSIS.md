# PathfinderSSH MSP — synopsis

**What it is:** A single Windows desktop app for network engineers — SSH/telnet/serial terminals, discovery crawl, config capture, topology maps, and an MSP-oriented session inventory — extended for managed service providers with customer folders, RMM/doc integrations, and optional cloud sign-in.

**What it is not:** A hosted SaaS, a replacement for your PSA, or upstream PathfinderSSH itself. This tree is a **GPL-3.0 derivative** maintained by AgenticOps (`pathfinder-lane`). The original engine and vision are Scott Peterman’s [PathfinderSSH](https://github.com/scottpeterman/pathfinderssh).

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

- **GUI installer** — `pfinstall.exe` → wizard: Solo / Microsoft 365 / Google
- **Solo** — no cloud OAuth; local enrollment only
- **Microsoft 365 / Google** — OIDC + PKCE in system browser; enrollment file + per-user session
- Packages: `internal/idp`, `internal/mspenroll`, `internal/mspauth`
- Tools: `pfinstall.exe`, `pathfinder -install-gui`, `cmd/pfenroll`

### MSP integrations (cohesive stack — cloud sign-in only)

Three roles merge in `Customers/<client>/`:

| Role | Systems | Outcome |
| --- | --- | --- |
| **Customers** | ConnectWise, Autotask, Halo, `psa-customers.json` | `Customers/<name>/` folders |
| **Inventory** | Auvik, Domotz, NinjaOne, Datto RMM, Automate, N-central | SSH sessions with IPs; re-sync updates addresses |
| **Credentials** | IT Glue, Hudu, Passportal | Vault passwords linked to sessions by host/name |

Pick **one per layer** — not every vendor. Inventory does not export SSH passwords; pair with a vault.

→ [MSP-INTEGRATION-STACK.md](MSP-INTEGRATION-STACK.md)

Excluded: audit-only platforms that do not feed live SSH targets (e.g. Liongard).

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
cmd/pfinstall/      Primary installer (GUI + CLI)
cmd/pfenroll/       Enrollment wizard only
cmd/pfvault/        Vault CLI
internal/auvik/     Auvik API client + sync
internal/itglue/    IT Glue API + vault/session link
internal/invsync/   Generic inventory → session tree
internal/docvault/  Generic vault import + session link
internal/mspsync/   Stack folder names and tags
internal/connectwise/ internal/autotask/ internal/halo/  PSA customers
internal/domotz/ internal/ninja/ internal/dattormm/ internal/automate/ internal/ncentral/  Inventory
internal/hudu/ internal/passportal/  Doc vault
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
| Tier-1 inventory (Domotz, Ninja, Datto, Automate, N-central) | **Shipped** (API clients; validate against live tenants) |
| Tier-1 vault (Hudu, Passportal) | **Shipped** |
| Tier-1 PSA (ConnectWise, Autotask, Halo) | **Shipped** |
| Cursor AI pane + troubleshoot addon | **Shipped** (addon gated) |
| JIT bastion, compliance cop, password rotation | **Roadmap** |

---

## Next reads

- [MSP-FEATURES.md](MSP-FEATURES.md) — where each feature lives in the UI
- [INTEGRATIONS.md](INTEGRATIONS.md) — middleware architecture
- [INSTALL.md](INSTALL.md) — get a build on your machine
