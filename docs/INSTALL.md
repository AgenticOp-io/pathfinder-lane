# Install and update

## Graphical install (recommended)

```powershell
cd products\pathfinder-msp
.\Install.ps1
```

Or launch the wizard from an existing binary:

```powershell
pathfinder -install-gui
```

![Install wizard](images/msp-install-wizard.svg)

Pick **Solo**, **Microsoft 365**, or **Google**. The installer copies `pathfinder.exe` to:

`%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pathfinder.exe`

Shortcuts are created under Start Menu → PathfinderSSH MSP.

## Silent / scripted install

```powershell
.\Install.ps1 -setup solo
.\Install.ps1 -setup o365
.\Install.ps1 -setup google
```

CLI after install:

```powershell
pathfinder -setup solo
pathfinder -setup o365
pathfinder -setup google
```

## Update

```powershell
cd products\pathfinder-msp
.\Update-Install.ps1
```

Preserves `%USERPROFILE%\.pathfinderssh\` data (sessions, vault, maps).

## Build from source

```bash
go build -ldflags "-s -w -H windowsgui" -o pathfinder.exe ./cmd/pathfinder
```

Windows requires `-H windowsgui` so no console window appears beside the GUI.

## Paths

| Path | Contents |
| --- | --- |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\` | Installed binary, enrollment JSON |
| `%USERPROFILE%\.pathfinderssh\` | sessions.yaml, vault, maps, scripts, settings |
| `%USERPROFILE%\.pathfinderssh\maps\<Customer>\` | Per-customer topology JSON |

## First launch

1. Unlock or create **vault** (all modes)
2. Optional: Settings → Tools → Auvik / IT Glue
3. Import SecureCRT or sync Auvik
4. Select a session → **Launch**

## Cloud sign-in setup (first time)

**Solo** — no cloud app registration. Enrollment: `%LOCALAPPDATA%\PathfinderSSH-MSP\msp-enrollment.json` (`provider: local`).

**Microsoft 365** (~5 minutes):

1. Azure → App registrations → New registration (`PathfinderSSH`, single tenant)
2. Redirect URI (Web): `http://127.0.0.1:53682/callback`
3. Authentication → Allow public client flows: **Yes**
4. Copy Tenant ID and Client ID into the install wizard or `pathfinder -setup o365`

**Google** (~5 minutes):

1. Google Cloud Console → Credentials → OAuth client ID (Desktop or Web + redirect above)
2. Copy Client ID into the wizard or `pathfinder -setup google`

Change mode later: delete `msp-enrollment.json` and re-run `pathfinder -setup solo|o365|google`. Sessions and vault under `~\.pathfinderssh` are kept.

Standalone wizard: `go run ./cmd/pfinstall` or `pathfinder -install-gui`.

Full auth detail: [AUTH.md](AUTH.md). Optional Entra script: `products/pathfinder-msp/deploy/entra/`.
