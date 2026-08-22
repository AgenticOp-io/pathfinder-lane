# MSP integration settings

All integration fields live in **Settings → Tools** (MSP cloud sign-in only). Solo enrollment hides this section.

See [MSP-INTEGRATION-COMBINATIONS.md](MSP-INTEGRATION-COMBINATIONS.md) for how settings connect to the three-layer stack.

---

## Inventory defaults (all RMM sources)

| Setting | Purpose |
| --- | --- |
| **Default SSH user** | Applied to new inventory sessions when username is empty |
| **Default vault credential** | Applied when credential is empty; sets `AuthType` to password |

These replace the legacy Auvik-only default fields. Auvik import still reads Auvik-specific defaults if MSP defaults are empty.

---

## Auvik

| Setting | Env var | Notes |
| --- | --- | --- |
| Username | `AUVIK_USERNAME` | API user email |
| API key | `AUVIK_API_KEY` | |
| Base URL | `AUVIK_BASE_URL` | Default `https://auvikapi.us1.my.auvik.com` |
| Periodic sync enabled | — | Background sync all tenants |
| Sync interval (minutes) | — | Minimum 5; default 60 |
| Auto-tunnel on import | — | Sets `AuvikUseTunnel` on new nodes |

Actions: **Import from Auvik**, **Sync Auvik now** (File menu + Tools).

---

## IT Glue

| Setting | Env var |
| --- | --- |
| API key | `ITGLUE_API_KEY` |
| Base URL | `ITGLUE_BASE_URL` |

API key requires **Password Access** enabled in IT Glue.

Action: **Import from IT Glue** (File menu).

---

## Hudu

| Setting | Env var |
| --- | --- |
| API key | `HUDU_API_KEY` |
| Base URL | `HUDU_BASE_URL` |

Action: **Import credentials from Hudu…**

---

## Passportal (N-able)

| Setting | Env var |
| --- | --- |
| API key | `PASSPORTAL_API_KEY` |
| Tenant | `PASSPORTAL_TENANT` |
| Base URL | `PASSPORTAL_BASE_URL` |

Action: **Import credentials from Passportal…**

---

## ConnectWise Manage (PSA)

| Setting | Env var |
| --- | --- |
| Company ID | `CONNECTWISE_COMPANY_ID` |
| Public key | `CONNECTWISE_PUBLIC_KEY` |
| Private key | `CONNECTWISE_PRIVATE_KEY` |
| Client ID | `CONNECTWISE_CLIENT_ID` |
| Base URL | `CONNECTWISE_BASE_URL` |

Action: **Sync customers from ConnectWise…**

---

## Datto Autotask (PSA)

| Setting | Env var |
| --- | --- |
| API username | `AUTOTASK_USERNAME` |
| Secret | `AUTOTASK_SECRET` |
| Integration code | `AUTOTASK_API_INTEGRATION_CODE` |
| Base URL | `AUTOTASK_BASE_URL` |

Action: **Sync customers from Autotask…**

---

## Halo PSA

| Setting | Env var |
| --- | --- |
| Client ID | `HALO_CLIENT_ID` |
| Client secret | `HALO_CLIENT_SECRET` |
| Tenant subdomain | `HALO_TENANT` |
| Base URL | `HALO_BASE_URL` |

Action: **Sync customers from Halo…**

---

## Domotz

| Setting | Env var |
| --- | --- |
| API key | `DOMOTZ_API_KEY` |
| Base URL | `DOMOTZ_BASE_URL` |

Action: **Sync devices from Domotz…**

---

## NinjaOne

| Setting | Env var |
| --- | --- |
| Client ID | `NINJA_CLIENT_ID` |
| Client secret | `NINJA_CLIENT_SECRET` |
| Base URL | `NINJA_BASE_URL` |

OAuth client credentials flow.

Action: **Sync devices from NinjaOne…**

---

## Datto RMM

| Setting | Env var |
| --- | --- |
| API key | `DATTO_API_KEY` |
| Secret | `DATTO_SECRET` |
| Base URL | `DATTO_BASE_URL` |

Action: **Sync devices from Datto RMM…**

---

## ConnectWise Automate

| Setting | Env var |
| --- | --- |
| Server URL | `AUTOMATE_SERVER_URL` |
| Username | `AUTOMATE_USERNAME` |
| Password | `AUTOMATE_PASSWORD` |

Action: **Sync devices from Automate…**

---

## N-able N-central

| Setting | Env var |
| --- | --- |
| JWT | `NCENTRAL_JWT` |
| Server URL | `NCENTRAL_SERVER_URL` |

Action: **Sync devices from N-central…**

---

## Security notes

- Settings stored in `%USERPROFILE%\.pathfinderssh\settings.json` (DPAPI on Windows where applicable)
- API keys are never logged or sent to AgenticOps
- Vault master password is separate from all integration credentials
- MSP cloud sign-in (Entra/Google) is separate from vendor API keys

See [AUTH.md](AUTH.md) and [SECURITY.md](../SECURITY.md).
