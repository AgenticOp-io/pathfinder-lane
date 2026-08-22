# MSP features — UI tour

Screenshots use **synthetic names only** (Contoso, Northwind, Fabrikam). See [images/README.md](images/README.md).

## Main window

![Main window](images/msp-main-window.svg)

| Area | Purpose |
| --- | --- |
| **Top toolbar** | Connect, Crawl, Capture, Map, Search, Scripts, Tabs, Settings |
| **Left tree** | `Customers/` and `Unassigned/` session inventory |
| **Filter** | Ctrl+L — filter by folder or session name |
| **Footer actions** | Launch, Add session, Add folder, Add user, Edit, Delete |
| **Status** | Total session count; hint when nothing is open |

## Customer tree

![Customer tree](images/msp-customer-tree.svg)

- One folder per customer under **Customers**
- Nested folders for site, vendor, or role (e.g. OOB, Servers)
- Leaf nodes are connectable sessions (SSH, telnet, serial)
- **Unassigned** holds imports not yet filed under a customer

Import paths:

- SecureCRT tree → **Customers** on first run
- Auvik sync → `Customers/<client>/Auvik/`
- PSA JSON → customer folder names
- CSV crawl seeds → [customer-crawl-seeds.example.csv](customer-crawl-seeds.example.csv)

## Active session

![Active SSH session](images/msp-session-active.svg)

| Element | Behavior |
| --- | --- |
| **Tab** | One tab per open session; Ctrl+W closes |
| **Session header** | Shows vault user and target host |
| **Terminal** | Full scrollback, themes, detach to window |
| **Button bar** | YAML macros — `send:` one-liners or `script:` files |
| **All tabs** | Arm macro to every open terminal |

Ops extras: SFTP tab, port forwards, tile layout, evidence pack, script recorder.

## Settings → Tools (integrations)

![Settings Tools](images/msp-settings-tools.svg)

Configure **Auvik** (inventory) and **IT Glue** (credentials). Details:

- [INTEGRATIONS.md](INTEGRATIONS.md)
- [AUVIK-API.md](AUVIK-API.md)
- [ITGLUE-API.md](ITGLUE-API.md)

## Install wizard

![Install wizard](images/msp-install-wizard.svg)

Graphical first-run and `Install.ps1 -install-gui`:

- **Solo** — local vault only
- **Microsoft 365** — Entra OIDC
- **Google** — Workspace OIDC

See [INSTALL.md](INSTALL.md) and [AUTH.md](AUTH.md).

## Cursor AI pane (optional)

![Cursor pane](images/msp-cursor-pane.svg)

Enable in Settings → Cursor. Troubleshoot modal sends session context to Cloud Agents API. See [CURSOR-AI.md](CURSOR-AI.md).

## Feature matrix

| Feature | Doc |
| --- | --- |
| Install / update | [INSTALL.md](INSTALL.md) |
| Sign-in modes | [AUTH.md](AUTH.md) |
| Auvik sync | [AUVIK-API.md](AUVIK-API.md) |
| IT Glue vault | [ITGLUE-API.md](ITGLUE-API.md) |
| Day-to-day workflows | [OPERATIONS.md](OPERATIONS.md) |
| Binaries / flags | [CLI.md](CLI.md) |
| Packages / data flow | [ARCHITECTURE.md](ARCHITECTURE.md) |
