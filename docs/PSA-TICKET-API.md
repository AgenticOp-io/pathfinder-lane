# PSA ticket documentation API

Engineer work documentation can bind and post to PSA tickets (ConnectWise Manage, Datotask, Halo) in addition to PagerDuty/Opsgenie.

## Bind work context

Settings → File → **Bind active incident…** → choose provider:

- `connectwise` — ticket id or ticket number
- `autotask` — ticket id or ticket number
- `halo` — ticket id

On bind, Pathfinder looks up the ticket via the PSA REST API and fills customer folder + title when omitted.

Credentials use the same fields as PSA customer sync (Settings → Integrations).

## Document work

**Document work to incident…** (session menu or Settings → File) builds the same capture pack as PagerDuty:

- Terminal scrollback(s)
- Optional customer topology map
- Optional running-config captures

Then posts to the bound provider:

| PSA | Note | Attachment |
| --- | --- | --- |
| ConnectWise | Ticket note (`detailDescriptionFlag`) | Document upload to ticket |
| Autotask | Ticket note | Local path in note (tenant attachment APIs vary) |
| Halo | Ticket action note | Local path in note |

A local zip is always saved under the app logs directory.

## Customer handoff export

Settings → File → **Export customer handoff package…** writes a zip with:

- `sessions.yaml` for the customer subtree (no vault secrets)
- `inventory-meta.json`
- `maps/*.json` when present

## Console fallback (SSH + serial)

Session form → **Console fallback** names a sibling session in the same folder. When SSH dial fails, Pathfinder retries the fallback session automatically.

## Python script catalog

Place `.py` files in:

`~/.pathfinderssh/python-scripts/` (or app home `python-scripts/`)

**Run Python script…** lists catalog scripts or browses for a file.

## NOC map view

Settings → File → **Open NOC map view…** opens the latest map JSON for a customer in the browser map viewer. Use **Reload** in the viewer (or F5) after crawls; fullscreen the browser tab for wallboard use. Team multi-user wallboards remain out of scope.
