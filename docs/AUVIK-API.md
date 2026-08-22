# Auvik integration

## What it does

| Action | Result |
| --- | --- |
| List tenants / devices | REST via `internal/auvik` |
| Sync tenant | Creates/updates sessions under `Customers/<client>/Auvik/` |
| Merge | Existing sessions matched by Auvik device ID; IP updated from Auvik |
| Periodic sync | Settings → Tools → enable interval for all linked clients |
| Auto tunnel | Launch `AuvikTunnel` when direct SSH fails (if binary configured) |

## Setup

1. Settings → **Tools** → Auvik
2. Enter tenant API key (from Auvik → Integrations → API)
3. File → **Sync from Auvik** or per-tenant sync from tree context menu

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
| SSH passwords from Auvik | ❌ not in API |
| Headless SSH via Auvik REST | ❌ use Pathfinder SSH or AuvikTunnel |

## Code references

- `internal/auvik/` — client, sync, merge
- Settings fields in `internal/ui/settings` (Tools tab)
- `importFromAuvik` / sync actions in `cmd/pathfinder/main.go`

See also [INTEGRATIONS.md](INTEGRATIONS.md) and [ROADMAP-FRONTEND.md](../ROADMAP-FRONTEND.md).
