# Opsgenie API — incident documentation for PathfinderSSH MSP

Pathfinder **augments** Opsgenie the same way as PagerDuty: bind an active alert, capture engineer work, and post a note back.

## Setup

1. Opsgenie → **Settings → Integrations → API** → create API key with alert read/write.
2. Pathfinder → **Settings → Tools**:
   - **Opsgenie API key**
   - **Opsgenie base URL** (default `https://api.opsgenie.com`)
3. Or environment: `OPSGENIE_API_KEY`, `OPSGENIE_BASE_URL`.

Requires MSP cloud sign-in (Microsoft 365 or Google).

## Workflow

Same as [PAGERDUTY-API.md](PAGERDUTY-API.md):

1. **Bind incident** — choose provider **opsgenie** in the bind dialog.
2. Ops desk filters the session tree to the customer folder.
3. **Document work to incident** — post-change capture pack (scrollback + map + configs) + incident note.

## API surface

| Call | Purpose |
| --- | --- |
| `GET /v2/alerts?limit=1` | Verify API key |
| `POST /v2/alerts/{id}/notes?identifierType=id` | Post engineer note |

Authorization: `GenieKey <api_key>`

## Code

`internal/opsgenie` implements `incidentbridge.Bridge` alongside `internal/pagerduty`.
