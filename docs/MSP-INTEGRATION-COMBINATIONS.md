# MSP integration combinations

This guide explains how **every supported PSA × inventory × vault combination** works in PathfinderSSH MSP. All combinations share the same customer tree, merge rules, and credential linking — not separate import silos.

**Prerequisite:** Microsoft 365 or Google enrollment (Solo hides integration UI). See [AUTH.md](AUTH.md).

**Architecture overview:** [MSP-INTEGRATION-STACK.md](MSP-INTEGRATION-STACK.md)

---

## The three layers (pick one per layer)

| Layer | Pick one | Creates / updates |
| --- | --- | --- |
| **Customers** | ConnectWise Manage, Datto Autotask, Halo PSA, or `psa-customers.json` | `Customers/<name>/` folder |
| **Inventory** | Auvik, Domotz, NinjaOne, Datto RMM, ConnectWise Automate, or N-able N-central | SSH sessions with **host/IP** under `Customers/<name>/<source>/` |
| **Credentials** | IT Glue, Hudu, or Passportal | Encrypted vault entries + optional session linking |

You do **not** need every vendor. A typical shop uses **one PSA + one RMM + one doc platform** they already pay for.

---

## Valid combination matrix

Every cell below is supported. Inventory sources can coexist under the same customer (e.g. Auvik + Domotz); vault imports link across **all** subfolders.

### Customers (PSA) — any one

| System | Settings → Tools | File menu |
| --- | --- | --- |
| ConnectWise Manage | Company ID, public/private keys, base URL | Sync customers from ConnectWise… |
| Datto Autotask | API user, secret, integration code | Sync customers from Autotask… |
| Halo PSA | OAuth client ID/secret, tenant | Sync customers from Halo… |
| JSON file | — | Import PSA customers (`psa-customers.json`) |

### Inventory — any one (or more on same customer)

| System | Folder under customer | File menu |
| --- | --- | --- |
| Auvik | `Auvik/` | Import / Sync from Auvik |
| Domotz | `Domotz/` | Sync devices from Domotz… |
| NinjaOne | `Ninja/` | Sync devices from NinjaOne… |
| Datto RMM | `DattoRMM/` | Sync devices from Datto RMM… |
| ConnectWise Automate | `Automate/` | Sync devices from Automate… |
| N-able N-central | `N-central/` | Sync devices from N-central… |

### Credentials — any one

| System | File menu |
| --- | --- |
| IT Glue | Import from IT Glue |
| Hudu | Import credentials from Hudu… |
| Passportal | Import credentials from Passportal… |

---

## Example end-to-end workflows

### ConnectWise + Auvik + IT Glue (common stack)

1. **File → Sync customers from ConnectWise…** → creates `Customers/Contoso Ltd/`
2. **File → Import from Auvik** → pick tenant `Contoso`; folder picker resolves to `Contoso Ltd` → devices in `Auvik/`
3. **File → Import from IT Glue** → pick org; folder picker resolves to `Contoso Ltd` → vault import + link sessions
4. **Launch** from tree — host from Auvik, password from vault

### Halo + NinjaOne + Hudu

1. Sync Halo customers
2. Sync NinjaOne devices → `Customers/<client>/Ninja/`
3. Import Hudu passwords → link under same customer folder
4. Launch

### Autotask + Datto RMM + Passportal

1. Sync Autotask customers
2. Sync Datto RMM devices → `DattoRMM/`
3. Import Passportal → pick customer folder for linking
4. Launch

### JSON PSA + Domotz + IT Glue (no live PSA API)

1. Import `psa-customers.json`
2. Sync Domotz per customer
3. Import IT Glue per org

### Mixed inventory on one customer

1. PSA sync creates `Customers/Northwind Traders/`
2. Auvik sync → `Northwind Traders/Auvik/` (network gear)
3. Later: Automate sync → `Northwind Traders/Automate/` (servers)
4. Same device IP in both sources → **one session** updated (merge by IP), moved to the folder of the latest sync source if needed
5. Vault import links credentials to sessions in **both** subfolders

---

## Customer folder alignment (critical for combinations)

PSA, Auvik tenants, and doc orgs often use slightly different names:

| Source | Label |
| --- | --- |
| ConnectWise | `Contoso Ltd` |
| Auvik tenant | `Contoso` |
| IT Glue org | `Contoso Inc` |

Pathfinder **fuzzy-resolves** these to one folder:

1. Exact match on folder name
2. Normalized match (case, punctuation)
3. Token overlap score (≥ 80%) — e.g. `Contoso` → `Contoso Ltd`

Resolution runs automatically during:

- Inventory sync (`invsync`, Auvik periodic sync)
- Vault session linking (handlers resolve before `LinkSessions`)

**UI:** Import dialogs show a **dropdown of existing customer folders** with the best match pre-selected. You can type a new folder name if the customer does not exist yet.

Implementation: `internal/mspsync.ResolveCustomerName`

---

## Inventory merge rules (all vendors)

Applies to Auvik, Domotz, Ninja, Datto, Automate, N-central equally.

### Match order (within `Customers/<client>/` and all subfolders)

1. **Scoped device ID** — `integration_source:external_device_id` (e.g. `domotz:abc123`)
2. **Management IP** — cross-vendor; Auvik session merges with Domotz on same IP
3. **Display name** — session label / device name

### On match

| Field | Behavior |
| --- | --- |
| **Host / IP** | Inventory source wins (updates on re-sync) |
| Username | Preserved if already set |
| Vault credential | Preserved |
| Terminal settings | Preserved |
| Auvik tunnel flag | Preserved / set on Auvik sync |

### Scoped IDs prevent false merges

Device ID `42` from NinjaOne does **not** match device ID `42` from Domotz on a different host. IP matching still unifies the same physical device across vendors.

### Shared defaults

Settings → Tools → **Default SSH user** and **Default vault credential** apply to all inventory syncs (including Auvik). Fills username/credential only when the session node is empty.

---

## Credential linking (all vaults)

After vault import, linking walks **every SSH session under the customer folder** — including `Auvik/`, `Domotz/`, manual site folders, etc.

### Match order

1. `external_password_id` or `itglue_password_id` on session node
2. Session **host** contained in password URL or resource URL
3. **Name** — password name contains session label, or vice versa

### Vault tags (re-import)

| Source | ID tag in vault |
| --- | --- |
| IT Glue | `itglue-id:<id>` |
| Hudu | `hudu-id:<id>` |
| Passportal | `passportal-id:<id>` |

Re-import with **update existing** refreshes passwords without duplicating vault entries.

---

## Settings reference (MSP cloud sign-in only)

Settings → **Tools** tab groups fields by role:

| Section | Fields |
| --- | --- |
| **Inventory defaults** | Default SSH user, default vault credential |
| **Auvik** | Username, API key, base URL, periodic sync, auto-tunnel |
| **IT Glue** | API key, base URL |
| **Hudu** | API key, base URL |
| **Passportal** | API key, tenant, base URL |
| **ConnectWise** | Company ID, public/private keys, client ID, base URL |
| **Autotask** | API user, secret, integration code, base URL |
| **Halo** | Client ID/secret, tenant, base URL |
| **Domotz** | API key, base URL |
| **NinjaOne** | Client ID/secret, base URL |
| **Datto RMM** | API key, secret, base URL |
| **Automate** | Server URL, username, password |
| **N-central** | JWT, server URL |

Environment variables mirror each field (see per-vendor `*-API.md` docs).

Solo enrollment: entire Tools integration section and File-tab sync actions are **hidden**.

---

## File menu actions (MSP cloud sign-in only)

| Action | Layer |
| --- | --- |
| Sync customers from ConnectWise… | PSA |
| Sync customers from Autotask… | PSA |
| Sync customers from Halo… | PSA |
| Import PSA customers | PSA (JSON) |
| Import from Auvik / Sync Auvik | Inventory |
| Sync devices from Domotz… | Inventory |
| Sync devices from NinjaOne… | Inventory |
| Sync devices from Datto RMM… | Inventory |
| Sync devices from Automate… | Inventory |
| Sync devices from N-central… | Inventory |
| Import from IT Glue | Credentials |
| Import credentials from Hudu… | Credentials |
| Import credentials from Passportal… | Credentials |

---

## What is intentionally excluded

Systems that do **not** feed the SSH loop are not integrated:

- Config-audit / snapshot-only platforms (e.g. Liongard-style inventories without live management IPs)
- PSA ticket APIs without customer folder mapping
- Password managers without MSP doc API (use vault manually)

Adding a vendor requires: live **IPs** or **passwords** that map to SSH sessions under `Customers/<client>/`.

---

## Troubleshooting combinations

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| Duplicate sessions for same device | Different customer folders (`Contoso` vs `Contoso Ltd`) | PSA sync first; use folder picker; names auto-resolve on re-sync |
| Inventory sync creates new session instead of updating IP | Device on different IP in source; or different customer folder | Re-sync after IP fix in RMM; verify customer folder |
| Vault import works but no sessions linked | Wrong customer folder for linking | Pick correct folder in import dialog; ensure inventory sync ran first |
| Solo install shows no integration settings | By design | Enroll with Microsoft 365 or Google |
| Auvik tunnel not used | `AuvikUseTunnel` not set | Enable auto-tunnel in Settings or per-session |
| Credential not applied on Launch | Session has no credential reference | Re-run vault import with linking; check host/name match |

---

## Code reference

| Package | Role |
| --- | --- |
| `internal/mspsync` | Stack definition, folder names, `ResolveCustomerName`, `DeviceIDKey` |
| `internal/invsync` | Generic RMM inventory → session tree |
| `internal/docvault` | Generic vault import + session linking |
| `internal/psasync` | PSA customer list → folders |
| `internal/auvik` | Auvik sync, tunnel, periodic sync |
| `internal/itglue` | IT Glue vault path (parallel to docvault) |
| `internal/mspauth/integrations.go` | `IntegrationsEnabled` gate |
| `internal/ui/mspcustomerpicker.go` | Customer folder dropdown + fuzzy pre-select |
| `cmd/pathfinder/mspintegrations.go` | File menu handlers |

Tests: `internal/mspsync/customer_test.go`, `internal/invsync/sync_test.go` (cross-source IP merge, scoped device IDs).

---

## Related docs

- [MSP-INTEGRATION-STACK.md](MSP-INTEGRATION-STACK.md) — architecture
- [INTEGRATIONS.md](INTEGRATIONS.md) — overview
- [OPERATIONS.md](OPERATIONS.md) — day-to-day checklist
- Per-vendor: [AUVIK-API.md](AUVIK-API.md), [ITGLUE-API.md](ITGLUE-API.md), [CONNECTWISE-API.md](CONNECTWISE-API.md), etc.
