# IT Glue API — credentials for PathfinderSSH MSP

Pathfinder pairs **Auvik** (device inventory / IPs) with **IT Glue** (username/password vault).

## Setup

1. IT Glue → **Admin → Settings → API Keys → Create API Key**
2. Enable **Password Access** on the key (required to read plaintext passwords)
3. Pathfinder → **Settings → Tools** → paste API key and base URL:
   - US: `https://api.itglue.com`
   - EU: `https://api.eu.itglue.com`
   - AU: `https://api.au.itglue.com`

Or set env `ITGLUE_API_KEY` (overrides settings when set).

## Import workflow

**Settings → File → Import credentials from IT Glue…**

1. Pick organization (maps to `Customers/<name>/`)
2. Passwords are fetched via show endpoint (secrets in memory only)
3. Stored in your **encrypted local vault** with `itglue-id:` tags
4. Optional: link vault entries to SSH sessions under that customer (by host/name match)

Recommended MSP flow:

1. **Auvik** — import/sync devices into `Customers/<client>/Auvik/`
2. **IT Glue** — import credentials and link to those sessions
3. Connect — techs use vault credentials without copying passwords from IT Glue UI

## Security

- API keys and passwords are **never written to logs**
- Plaintext exists only in memory during import, then in the encrypted vault file
- Rate limits: batch imports use pagination; re-import updates existing `itglue-id` entries

## Architecture (middleware)

| Source | Role in Pathfinder |
|--------|-------------------|
| Auvik API | Device discovery, IP authority, optional AuvikTunnel |
| IT Glue API | Credential vaulting, session linking |
| Local vault | Encrypted storage; SSH dialer reads credentials at connect time |

Future: JIT bastion, compliance audits, and rotation can extend this client without changing the vault model.
