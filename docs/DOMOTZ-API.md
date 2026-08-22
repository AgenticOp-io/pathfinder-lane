# Domotz integration

**Role:** Inventory — device IPs and new gear for SSH.

Sync creates/updates sessions under `Customers/<client>/Domotz/`. Re-sync updates IPs when Domotz reports changes.

## Setup

1. Settings → Tools → Domotz (MSP cloud sign-in required)
2. Enter API key from Domotz → Settings → API
3. File → **Sync devices from Domotz…** → enter customer folder name

Env: `DOMOTZ_API_KEY`, `DOMOTZ_BASE_URL`

## Limits

- Provides management IPs for accepted devices
- Does not export SSH passwords — pair with IT Glue, Hudu, or Passportal

Part of the [MSP integration stack](MSP-INTEGRATION-STACK.md).
