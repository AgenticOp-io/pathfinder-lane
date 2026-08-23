# Auvik integration

## What it does

| Action | Result |
| --- | --- |
| List tenants / devices | REST via `internal/auvik` |
| Sync tenant | Creates/updates sessions under `Customers/<client>/Auvik/` |
| Merge | Existing sessions matched by Auvik device ID; IP updated from Auvik |
| Tenant → customer map | `auvik-tenant-map.json` persists folder overrides per tenant id |
| Periodic sync | Settings → Tools → enable interval; runs immediately on start + every N minutes |
| Headless sync | `pfseed sync-auvik` for Task Scheduler / unattended estates |
| Stale prune | Optional: remove Auvik sessions missing from inventory on sync |
| Auto tunnel | Launch `AuvikTunnel` when direct SSH fails (if binary configured) |

## Setup

1. Settings → **Tools** → Auvik (API key, periodic sync, prune stale, tunnel path)
2. Settings → **File** → **Sync devices from Auvik…** or **Sync all Auvik tenants now…**
3. Import saves tenant id → customer folder mapping for automated sync

## Headless automation

```bash
pfseed sync-auvik -sessions ~/.pathfinderssh/sessions.yaml -prune
```

Reads credentials from `settings.json` (or `-settings`). Use Windows Task Scheduler to run on an interval.

## Session layout

```
Customers/
  Contoso/
    Auvik/
      switch-core-01
      router-edge-01
```

Names come from Auvik device labels. Folder structure is flat under `Auvik/` unless you reorganize locally (local-only moves are preserved on merge).

## Limits (API reality)

| Capability | Status |
| --- | --- |
| Device IPs, names, IDs | ✅ |
| SSH passwords from Auvik | ❌ not in API — import IT Glue / Hudu / Passportal after sync |
| Headless SSH via Auvik REST | ❌ use Pathfinder SSH or AuvikTunnel |

## Code references

- `internal/auvik/` — client, sync, tenant map, stale prune, tunnel
- `cmd/pfseed sync-auvik` — unattended sync
- Settings fields in `internal/ui/mspintegrationpanel.go` (Tools tab)
- Sync actions in Settings → File (`cmd/pathfinder/main.go`)

See also [INTEGRATIONS.md](INTEGRATIONS.md) and [ROADMAP-FRONTEND.md](../ROADMAP-FRONTEND.md).
