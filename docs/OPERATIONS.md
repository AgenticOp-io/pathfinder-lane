# MSP day-to-day operations

## Morning checklist

1. Launch PathfinderSSH MSP (Start Menu shortcut)
2. Unlock vault
3. Optional: confirm Auvik periodic sync ran (Settings → Tools)
4. Filter tree (Ctrl+L) for today’s customer

## Onboarding a new customer

1. Create folder under **Customers** (or import from PSA JSON)
2. Sync Auvik tenant → populates `Auvik/` devices
3. IT Glue import → fills vault passwords and links sessions
4. File missing sessions manually or via CSV crawl seeds
5. Run crawl from seed host → import map under `maps/<Customer>/`

## Working a ticket

1. Launch session from tree
2. Use button bar macros (`show run`, `show lldp`, …)
3. Session menu → evidence pack for ticket attachment
4. Optional: Cursor troubleshoot pane for complex issues

## File layout conventions

```
Customers/
  Contoso/
    HQ/
    OOB/
    Auvik/          # synced — avoid manual edits to IPs
Unassigned/         # triage imports here, drag to customer when known
```

## Offboarding an engineer

| Mode | Action |
| --- | --- |
| Solo | Rotate vault password; remove Windows login |
| Entra / Google | Disable account in IdP; revoke app tokens |

## Imports

| Source | Menu / action |
| --- | --- |
| SecureCRT | First-run wizard or File → Import |
| Auvik | Settings / File → Sync |
| IT Glue | Settings → Tools → Import passwords |
| Crawl seeds CSV | File → Import customer crawl seeds |

Example CSV: [customer-crawl-seeds.example.csv](customer-crawl-seeds.example.csv)
