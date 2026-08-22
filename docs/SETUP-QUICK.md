# Quick setup — solo, Microsoft 365, or Google

Non-commercial / personal use: **Solo is the default** — no cloud app registration.

## Install

```powershell
cd products\pathfinder-msp
.\Install.ps1
```

Pick **1** for solo, **2** for Microsoft 365, **3** for Google.

## Solo (just me)

- No Entra or Google OAuth setup.
- Vault password keeps SSH credentials on your PC.
- Enrollment file: `%LOCALAPPDATA%\PathfinderSSH-MSP\msp-enrollment.json` (`provider: local`).

## Microsoft 365 (about 5 minutes once)

1. Azure → **App registrations** → **New registration**
2. Name: `PathfinderSSH`, single tenant
3. Redirect URI (Web): `http://127.0.0.1:53682/callback`
4. **Authentication** → Allow public client flows: **Yes**
5. Copy **Tenant ID** and **Client ID** into the Pathfinder setup wizard

Or run `.\Install.ps1 -Setup o365` and paste IDs when prompted.

## Google (about 5 minutes once)

1. Google Cloud Console → **Credentials**
2. **Create credentials** → OAuth client ID → Desktop (or Web + redirect above)
3. Copy **Client ID** into the Pathfinder setup wizard

Or run `.\Install.ps1 -Setup google`.

## Change mode later

Delete `%LOCALAPPDATA%\PathfinderSSH-MSP\msp-enrollment.json` and run:

```powershell
pathfinder.exe -setup solo
pathfinder.exe -setup o365
pathfinder.exe -setup google
```

Sessions and vault under `~\.pathfinderssh` are kept.
