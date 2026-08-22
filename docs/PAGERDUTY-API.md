# PagerDuty API — incident documentation for PathfinderSSH MSP

Pathfinder **augments** PagerDuty for the engineer on the keyboard. PSA/RMM/incident workflow stays in those products; this console binds an active incident, captures scrollback, and posts an engineer work note back to PagerDuty.

## Setup

1. PagerDuty → **Integrations → API Access Keys** → create a key with incident read/write scope.
2. Pathfinder → **Settings → Tools**:
   - **PagerDuty API key**
   - **PagerDuty base URL** (default `https://api.pagerduty.com`)
3. Or set environment variables:
   - `PAGERDUTY_API_KEY`
   - `PAGERDUTY_BASE_URL` (optional)

Requires **MSP cloud sign-in** (Microsoft 365 or Google). Solo installs hide MSP integrations.

## Engineer workflow

### Bind active incident

**Settings → File → Bind active incident…**

- Paste a PagerDuty incident id or URL.
- Optionally pick a **customer folder** (aligns with `Customers/<name>/`).
- Optional title and engineer notes for local context.

The bottom status bar shows `Incident <id> · <customer>` while bound. Context persists in `~/.pathfinderssh/work-context.json`.

**Clear active incident** removes the binding without posting to PagerDuty.

### Document work

**Settings → File → Document work to PagerDuty…**  
or **Session menu → Document work to PagerDuty…**

1. Confirm incident id (defaults to bound incident).
2. Optional note describing what you did.
3. Choose whether to include **all open terminal scrollbacks** or only the current tab.

Pathfinder:

1. Builds a plaintext **engineer summary** (incident, customer, hosts touched, open tabs, your note).
2. Saves a local **evidence zip** under the logs directory (`pathfinder-evidence-<id>-<timestamp>.zip`).
3. Posts an **incident note** via `POST /incidents/{id}/notes`.

PagerDuty REST notes **do not accept file uploads**. The note references the local zip path and byte size; attach the zip in PagerDuty or your PSA if auditors need the file in the ticket system.

## API surface used

| Call | Purpose |
| --- | --- |
| `GET /incidents?limit=1` | Verify API key on document |
| `POST /incidents/{id}/notes` | Post engineer work note |

Authorization: `Token token=<api_key>`  
Accept: `application/vnd.pagerduty+json`

## Architecture

| Package | Role |
| --- | --- |
| `internal/workcontext` | Local active incident binding + summary builder |
| `internal/incidentbridge` | Generic bridge interface (`Verify`, `PostDocument`) |
| `internal/pagerduty` | PagerDuty REST client + bridge |
| `internal/evidence` | Scrollback zip builder |
| `cmd/pathfinder/workdocument.go` | UI handlers, collect scrollback, post note |

**Phase 2 (planned):** Opsgenie adapter via the same `incidentbridge` interface; richer post-change capture packs (config + map) tied to incident id.

## Security

- API keys live in `settings.json` or env — never logged.
- Scrollback may contain secrets; review before documenting. Pathfinder does not scrub terminal output.
- Evidence zips stay on the engineer desktop until you attach them elsewhere.

See [MSP-INTEGRATION-STACK.md](MSP-INTEGRATION-STACK.md) (augment lane) and [MSP-FEATURE-GAPS.md](MSP-FEATURE-GAPS.md).
