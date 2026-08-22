# Integrations — middleware stack

PathfinderSSH MSP treats **inventory** and **credentials** as separate layers that merge in the local session tree.

```
┌─────────────┐     ┌─────────────┐
│   Auvik     │     │  IT Glue    │
│  (IPs,      │     │ (passwords, │
│   tenants)  │     │  org names) │
└──────┬──────┘     └──────┬──────┘
       │                   │
       ▼                   ▼
┌──────────────────────────────────┐
│  PathfinderSSH MSP (local)       │
│  Customers/<client>/… sessions   │
│  Encrypted vault (AES-256-GCM)   │
└──────────────────────────────────┘
```

## Auvik — inventory authority

- Sync devices into `Customers/<client>/Auvik/`
- Merge updates IPs on existing sessions (Auvik wins for address)
- Optional **AuvikTunnel** when SSH is only reachable via collector
- **Does not** export device SSH passwords via API

→ [AUVIK-API.md](AUVIK-API.md)

## IT Glue — credential authority

- Import organization passwords into encrypted vault
- Link sessions by hostname / flexible name match
- `ITGluePasswordID` stored on session nodes for re-sync

→ [ITGLUE-API.md](ITGLUE-API.md)

## PSA customers (file adapter)

- Import `psa-customers.json` to create customer folders
- Live ConnectWise / Autotask / Halo SDKs — roadmap (file adapter ships today)

## Settings UI

![Settings Tools](images/msp-settings-tools.svg)

Settings → **Tools** tab: API keys, sync buttons, periodic Auvik sync.

## Security notes

- API keys live in local settings (DPAPI on Windows where applicable)
- Vault master password is never sent to Auvik or IT Glue
- Sign-in (Entra / Google) is separate from RMM/doc API keys — see [AUTH.md](AUTH.md)
