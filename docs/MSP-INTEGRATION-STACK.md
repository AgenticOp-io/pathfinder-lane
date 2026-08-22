# MSP integration stack

PathfinderSSH MSP connects external systems through **four augment lanes** around the engineer SSH loop. Solo installs (local sign-in) hide this entire stack in Settings.

## The cohesive loop

```
  PSA customers          Inventory (IPs)         Credentials (vault)     Incident (work doc)
  ─────────────          ─────────────────         ───────────────────     ───────────────────
  ConnectWise            Auvik                     IT Glue                 PagerDuty
  Autotask               Domotz                    Hudu                    (Opsgenie planned)
  Halo PSA               NinjaOne                  Passportal
  JSON file              Datto RMM
                         Automate
                         N-central
              │                    │                        │                      │
              └──────────► Customers/<client>/ ◄────────────┘                      │
                           ├── <source>/devices…                                   │
                           └── sessions link to vault                               │
                                    │                                                │
                                    ▼                                                │
                              SSH Launch ──► document engineer work ───────────────┘
```

| Layer | What it gives Pathfinder | What happens on re-sync |
| --- | --- | --- |
| **Customers** | `Customers/<name>/` folder | New PSA clients appear as folders |
| **Inventory** | SSH sessions with **host/IP** | IPs update; new devices added; merge by device id / IP / name |
| **Credentials** | Username/password in **encrypted vault** | Vault entries update; sessions link by host/name |
| **Incident doc** | Bind active incident; post engineer notes + scrollback summary | Local work context; note posted to PagerDuty on document |

Systems that do **not** strengthen this loop are intentionally excluded (e.g. config-audit snapshots without live management IPs). Pathfinder **augments** PSA/RMM/incident apps — it does not replace ticket workflow or on-call routing.

Systems that do **not** strengthen this loop are intentionally excluded (e.g. config-audit snapshots without live management IPs).

**Full combination guide:** [MSP-INTEGRATION-COMBINATIONS.md](MSP-INTEGRATION-COMBINATIONS.md)  
**Settings reference:** [MSP-INTEGRATION-SETTINGS.md](MSP-INTEGRATION-SETTINGS.md)

## Who sees integrations

| Enrollment | Settings → Tools | Settings → File (MSP sync) |
| --- | --- | --- |
| **Solo** (Just me) | Hidden | Hidden |
| **Microsoft 365 / Google** | Full MSP stack | Full MSP sync buttons |

Cloud MSP sign-in is required: `mspauth.IntegrationsEnabled()` → Entra or Google enrollment.

## Recommended workflow

1. **Sync customers** — ConnectWise, Autotask, Halo, or `psa-customers.json`
2. **Sync inventory** — pick **one** RMM/monitoring source per client (Auvik is common; Domotz/Ninja/Datto/Automate/N-central are alternatives)
3. **Import credentials** — IT Glue, Hudu, or Passportal into vault; link sessions
4. **Bind incident** (optional) — PagerDuty id/URL when working an on-call item
5. **Launch** — dialer reads tree host + vault credential
6. **Document work** — post scrollback summary + local evidence zip reference to PagerDuty when done

Repeat inventory + credential sync on a schedule (Auvik supports periodic sync in Settings).

## Customer folder alignment

PSA companies, Auvik tenants, and doc org names often differ slightly (`Contoso` vs `Contoso Ltd`). Pathfinder:

1. **Fuzzy-resolves** external labels to existing `Customers/<name>/` folders (`mspsync.ResolveCustomerName`)
2. **Picker UI** — inventory and vault imports show existing folders with the best match pre-selected
3. **Re-sync** — periodic Auvik sync resolves tenant names against PSA folders automatically

Pick the same customer folder when linking vault credentials to inventory sessions.

## Inventory merge rules (all RMM sources)

Sessions under `Customers/<client>/` and **all subfolders** are indexed together. Match order:

1. **Scoped device ID** — `source:external_device_id` (e.g. `ninja:42` ≠ `domotz:42`)
2. Management **IP** — **cross-vendor**; same IP merges Auvik + Domotz + Ninja on one session
3. Display **name**

Username, vault credential, terminal settings, and Auvik tunnel flags on matched nodes are **preserved**. Inventory wins on IP changes.

Shared defaults: Settings → Tools → **Default SSH user** / **Default vault credential** (also used by Auvik).

## Credential linking

After vault import, sessions under the same customer folder are linked when:

- `external_password_id` already set on the node
- Session **host** appears in password URL / resource URL
- Session **label** matches password name

Tags in vault: `itglue-id:`, `hudu-id:`, `passportal-id:` for re-import.

## Per-system reference

| System | Role | Folder / action | Doc |
| --- | --- | --- | --- |
| Auvik | Inventory + tunnel | `Auvik/` | [AUVIK-API.md](AUVIK-API.md) |
| Domotz | Inventory | `Domotz/` | [DOMOTZ-API.md](DOMOTZ-API.md) |
| NinjaOne | Inventory | `Ninja/` | [NINJA-API.md](NINJA-API.md) |
| Datto RMM | Inventory | `DattoRMM/` | [DATTO-RMM-API.md](DATTO-RMM-API.md) |
| Automate | Inventory | `Automate/` | [AUTOMATE-API.md](AUTOMATE-API.md) |
| N-central | Inventory | `N-central/` | [NCENTRAL-API.md](NCENTRAL-API.md) |
| IT Glue | Credentials | vault + link | [ITGLUE-API.md](ITGLUE-API.md) |
| Hudu | Credentials | vault + link | [HUDU-API.md](HUDU-API.md) |
| Passportal | Credentials | vault + link | [PASSPORTAL-API.md](PASSPORTAL-API.md) |
| ConnectWise Manage | Customers | PSA sync | [CONNECTWISE-API.md](CONNECTWISE-API.md) |
| Datto Autotask | Customers | PSA sync | [AUTOTASK-API.md](AUTOTASK-API.md) |
| Halo PSA | Customers | PSA sync | [HALO-API.md](HALO-API.md) |
| PagerDuty | Incident doc | bind + note | [PAGERDUTY-API.md](PAGERDUTY-API.md) |

## Code map

| Package | Role |
| --- | --- |
| `internal/invsync` | Generic inventory → session tree |
| `internal/docvault` | Generic passwords → vault + link |
| `internal/mspsync` | Folder names, tags, customer resolve, scoped device IDs |
| `internal/workcontext` | Active incident binding + engineer summary |
| `internal/incidentbridge` | Generic incident documentation bridge |
| `internal/pagerduty` | PagerDuty REST client |
| `internal/evidence` | Scrollback evidence zip |
| `internal/psasync` | PSA customer → folder |
| `internal/auvik` | Auvik-specific sync + tunnel |
| `internal/itglue` | IT Glue (existing vault path) |
| `cmd/pathfinder/mspintegrations.go` | UI handlers |

Tests: `internal/mspsync/customer_test.go`, `internal/invsync/sync_test.go`.

## Security

- API keys in `settings.json` or environment variables — never logged
- Vault master password never sent to cloud APIs
- MSP cloud sign-in (Entra/Google) is separate from RMM/doc API keys

See [AUTH.md](AUTH.md) and [SECURITY.md](../SECURITY.md).
