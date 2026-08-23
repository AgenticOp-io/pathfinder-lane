# Install and update

**Primary installer:** `pfinstall.exe` — graphical wizard or command-line flags. No PowerShell required.

Download or build the Windows bundle (`pathfinder.exe`, `pfseed.exe`, `pfinstall.exe`, `pfenroll.exe` in one folder), then run the installer from that folder or pass `-from`.

## Graphical install

Double-click `pfinstall.exe`, or:

```cmd
pfinstall.exe
pfinstall.exe -install-gui
pfinstall.exe -install-gui -setup o365
```

After a prior install: Start Menu → **Install PathfinderSSH**.

## Command-line install

```cmd
pfinstall.exe -install
pfinstall.exe -install -setup solo
pfinstall.exe -update
pfinstall.exe -install -setup o365 -enroll
pfinstall.exe -from C:\path\to\bundle -install
pfinstall.exe -uninstall
pfinstall.exe -version
```

| Flag | Purpose |
| --- | --- |
| `-install` | Copy bundle to AppData, create shortcuts |
| `-update` | Refresh installed binaries (keeps user data) |
| `-install-gui` | Graphical wizard |
| `-uninstall` | Remove AppData install and shortcuts |
| `-setup solo\|o365\|google` | Configure access mode during install |
| `-enroll` | Complete cloud OAuth during CLI install (opens browser) |
| `-from <dir>` | Bundle folder (default: directory containing `pfinstall.exe`) |
| `-version` | Print installer version |

Cloud sign-in without `-enroll` installs binaries only; finish sign-in in the GUI wizard or app.

**Update** (rebuild from source, then reinstall):

```cmd
build-windows.ps1 -Targets pathfinder,pfseed,pfinstall,pfenroll
dist\windows\pfinstall.exe -update
```

Or one step after build:

```cmd
build-windows.ps1 -Targets pathfinder,pfseed,pfinstall,pfenroll -Install -Setup solo
```

Preserves `%USERPROFILE%\.pathfinderssh\` data (sessions, vault, maps).

## Alternate entry points

`pathfinder.exe` also accepts `-install`, `-install-gui`, `-uninstall` for convenience when the main app binary is already on PATH. Prefer `pfinstall.exe` for distribution and scripting.

## Installed layout

| Path | Contents |
| --- | --- |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pathfinder.exe` | Main app |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pfseed.exe` | Headless seeds / Auvik sync |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pfinstall.exe` | Installer |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pfenroll.exe` | Org enrollment (super admin) |

Shortcuts: **PathfinderSSH MSP** (app), **Install PathfinderSSH** (wizard).

## Build from source

```powershell
.\build-windows.ps1 -Targets pathfinder,pfseed,pfinstall,pfenroll
```

`pfinstall.exe` is built as a console-capable binary (CLI output works from `cmd.exe` and PowerShell). Fyne GUI apps use `-H windowsgui` except the installer.

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
4. Copy Tenant ID and Client ID into the install wizard or `pfinstall.exe -install-gui -setup o365`

**Google** (~5 minutes):

1. Google Cloud Console → Credentials → OAuth client ID (Desktop or Web + redirect above)
2. Copy Client ID into the wizard or `pfinstall.exe -install-gui -setup google`

Change mode later: delete `msp-enrollment.json` and re-run `pfinstall.exe -install -setup solo|o365|google`. Sessions and vault under `~\.pathfinderssh` are kept.

Full auth detail: [AUTH.md](AUTH.md).
