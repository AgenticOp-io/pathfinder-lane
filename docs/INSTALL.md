# Install and update

## Graphical install (recommended)

```powershell
cd engines\pathfinderssh
.\Install.ps1
```

Or double-click **Install PathfinderSSH** in Start Menu after a prior install.

Other graphical entry points:

```powershell
.\dist\windows\pfinstall.exe
.\dist\windows\pfinstall.exe -install-gui -setup o365
pathfinder -install-gui
```

## Silent / command-line install

```powershell
# Build installer bundle (pathfinder, pfseed, pfinstall, pfenroll)
.\build-windows.ps1 -Targets pathfinder,pfseed,pfinstall,pfenroll

# Silent install to %LOCALAPPDATA%\PathfinderSSH-MSP\bin\
.\dist\windows\pfinstall.exe -install
.\dist\windows\pfinstall.exe -install -setup solo
.\Install.ps1 -Setup solo

# Same via pathfinder binary
pathfinder -install
pathfinder -install -setup solo
```

Cloud sign-in during CLI install (opens browser):

```powershell
pfinstall.exe -install -setup o365 -enroll
```

Uninstall:

```powershell
pfinstall.exe -uninstall
.\Install.ps1 -Uninstall
pathfinder -uninstall
```

## Update (rebuild + reinstall)

```powershell
.\Update-Install.ps1
.\Update-Install.ps1 -Setup solo
.\build-windows.ps1 -Targets pathfinder,pfseed,pfinstall,pfenroll -Install
```

Preserves `%USERPROFILE%\.pathfinderssh\` data (sessions, vault, maps).

## Installed layout

| Path | Contents |
| --- | --- |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pathfinder.exe` | Main app |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pfseed.exe` | Headless seeds / Auvik sync |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pfinstall.exe` | Installer wizard |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pfenroll.exe` | Org enrollment (super admin) |

Shortcuts: **PathfinderSSH MSP** (app), **Install PathfinderSSH** (wizard).

## Build from source

```powershell
.\build-windows.ps1 -Targets pathfinder,pfseed,pfinstall,pfenroll
```

GUI apps use `-H windowsgui` automatically (no extra console window).

## User data paths

| Path | Contents |
| --- | --- |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\` | Install root, enrollment JSON |
| `%USERPROFILE%\.pathfinderssh\` | sessions.yaml, vault, maps, settings |
| `%USERPROFILE%\.pathfinderssh\maps\<Customer>\` | Per-customer topology JSON |

## First launch

1. Unlock or create **vault** (all modes)
2. Optional: Settings → Tools → Auvik / IT Glue
3. Import SecureCRT or sync Auvik
4. Select a session → connect

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

Change mode later: delete `msp-enrollment.json` and re-run install with `-setup solo|o365|google`. Sessions and vault under `~\.pathfinderssh` are kept.

Full auth detail: [AUTH.md](AUTH.md).
