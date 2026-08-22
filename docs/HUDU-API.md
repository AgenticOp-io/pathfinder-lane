# Hudu integration

**Role:** Credentials — username/password into encrypted vault.

Pair with any inventory source (Auvik, Domotz, etc.): inventory supplies IPs; Hudu supplies login.

## Setup

1. Settings → Tools → Hudu API key
2. File → **Import credentials from Hudu…**
3. Pick company → vault import → optional session linking under `Customers/<company>/`

Env: `HUDU_API_KEY`, `HUDU_BASE_URL`

See [MSP-INTEGRATION-STACK.md](MSP-INTEGRATION-STACK.md).
