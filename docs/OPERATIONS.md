# MSP day-to-day operations

## Morning checklist

1. Launch PathfinderSSH MSP (Start Menu shortcut)
2. Unlock vault
3. Optional: confirm Auvik periodic sync ran (Settings → Tools)
4. Filter tree (Ctrl+L) for today’s customer

## Onboarding a new customer

1. **Sync customers** from PSA (ConnectWise, Autotask, Halo) or import `psa-customers.json`
2. **Sync inventory** — pick one source (Auvik, Domotz, NinjaOne, Datto RMM, Automate, N-central)
3. **Import credentials** from IT Glue, Hudu, or Passportal → vault + session linking
4. Add crawl seeds or manual sessions for gaps
5. Run crawl from seed host → import map under `maps/<Customer>/`

See [MSP-INTEGRATION-STACK.md](MSP-INTEGRATION-STACK.md).

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

## Imports (MSP cloud sign-in required)

| Layer | Source | Action |
| --- | --- | --- |
| Customers | ConnectWise / Autotask / Halo | File → Sync customers… |
| Customers | `psa-customers.json` | File → Import PSA customers |
| Inventory | Auvik / Domotz / Ninja / Datto / Automate / N-central | File → Sync devices… |
| Credentials | IT Glue / Hudu / Passportal | File → Import credentials… |
| Legacy | SecureCRT | First-run wizard or File → Import |
| Crawl | CSV seeds | File → Import customer crawl seeds |

Example CSV: [customer-crawl-seeds.example.csv](customer-crawl-seeds.example.csv)
