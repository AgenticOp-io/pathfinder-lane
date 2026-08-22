# Integrations — cohesive MSP stack

PathfinderSSH MSP wires external systems into **one SSH loop**: customer folders → device IPs → vault credentials → Launch.

**MSP cloud sign-in only** (Microsoft 365 or Google). Solo installs hide all integration settings and File-tab sync actions.

→ Full stack guide: [**MSP-INTEGRATION-STACK.md**](MSP-INTEGRATION-STACK.md)  
→ All combinations: [**MSP-INTEGRATION-COMBINATIONS.md**](MSP-INTEGRATION-COMBINATIONS.md)

```
  PSA (customers)          Inventory (IPs)           Credentials (vault)
  ───────────────          ───────────────           ───────────────────
  ConnectWise              Auvik                     IT Glue
  Autotask                 Domotz                    Hudu
  Halo PSA                 NinjaOne                  Passportal
  psa-customers.json       Datto RMM
                           Automate
                           N-central
              │                    │                        │
              └──────────► Customers/<client>/ ◄────────────┘
                                    │
                                    ▼
                              SSH Launch
```

## Three roles — pick one per layer

| Role | Purpose | Options |
| --- | --- | --- |
| **Customers** | `Customers/<name>/` folders | ConnectWise, Autotask, Halo, JSON file |
| **Inventory** | Sessions with **host/IP**; merge on re-sync | Auvik, Domotz, NinjaOne, Datto RMM, Automate, N-central |
| **Credentials** | Username/password in vault; link to sessions | IT Glue, Hudu, Passportal |

You do **not** need every vendor — pick **one inventory + one vault + one PSA** that your shop already uses.

## Inventory merge (all RMM sources)

Re-sync matches existing sessions by external device id → IP → name. **Inventory wins on IP changes.** Username, vault credential, and terminal settings are preserved.

Shared defaults: Settings → Tools → **Default SSH user** / **Default vault credential**.

## What we exclude

Systems that do not feed live SSH targets or vault passwords stay out of the product (e.g. config-audit-only platforms). No fragmented “import anything” menu.

## Per-vendor setup

| System | Doc |
| --- | --- |
| Auvik | [AUVIK-API.md](AUVIK-API.md) |
| Domotz | [DOMOTZ-API.md](DOMOTZ-API.md) |
| NinjaOne | [NINJA-API.md](NINJA-API.md) |
| Datto RMM | [DATTO-RMM-API.md](DATTO-RMM-API.md) |
| Automate | [AUTOMATE-API.md](AUTOMATE-API.md) |
| N-central | [NCENTRAL-API.md](NCENTRAL-API.md) |
| IT Glue | [ITGLUE-API.md](ITGLUE-API.md) |
| Hudu | [HUDU-API.md](HUDU-API.md) |
| Passportal | [PASSPORTAL-API.md](PASSPORTAL-API.md) |
| ConnectWise Manage | [CONNECTWISE-API.md](CONNECTWISE-API.md) |
| Autotask | [AUTOTASK-API.md](AUTOTASK-API.md) |
| Halo PSA | [HALO-API.md](HALO-API.md) |

## Settings UI

![Settings Tools](images/msp-settings-tools.svg)

Settings → **Tools** tab (MSP only): API keys per vendor, periodic Auvik sync.

File tab: **Sync customers…**, **Sync devices…**, **Import credentials…** per connected system.

## Security

- API keys in local settings or environment variables
- Vault master password never sent to cloud APIs
- MSP sign-in (Entra/Google) is separate from RMM/doc API keys — see [AUTH.md](AUTH.md)
