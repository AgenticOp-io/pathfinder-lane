# Install and update

PathfinderSSH MSP has **two install paths**:

| Who | Tool | What it does |
| --- | --- | --- |
| **MSP admin / IT** | `pfinstall.exe` | Copies the full admin bundle to AppData (pathfinder, pfseed, setup tools) |
| **MSP admin / IT** | `pfsetup-o365.exe` or `pfsetup-google.exe` | **Full MSP setup:** branding → cloud auth → security policy → API integrations → engineer package |
| **Engineer** | `*-Engineer-Install.exe` (from admin package) | **Standalone MSP client:** branding, sign-in, security, APIs pre-applied — **no** admin auth or security tools |

## IT / admin install (`pfinstall.exe`)

| Action | Command |
| --- | --- |
| Graphical install | Double-click `pfinstall.exe` or `pfinstall.exe -install-gui` |
| Silent install | `pfinstall.exe -install` |
| Solo mode (CLI) | `pfinstall.exe -install -setup solo` |
| Update binaries | `pfinstall.exe -update` |
| Uninstall | `pfinstall.exe -uninstall` |

Flow: **Ready → Install → Complete**. Optional Solo checkbox (local vault + standalone MSP stack). After install, run **full MSP setup** (M365/Google) or **standalone MSP setup** (`pfsetup-apis`) for Auvik, PSA, and Cursor AI without cloud sign-in.

## Full MSP admin setup (`pfsetup-o365.exe` / `pfsetup-google.exe`)

Run once per tenant (super admin). Six-step wizard:

1. **Branding** — org name, product title, logo (`msp-branding.json`, `logo.png`)
2. **Cloud authentication** — Entra or Google OAuth app registration
3. **Verify tenant sign-in** — saves `msp-enrollment.json`
4. **Security policy** — read-only mode, change windows, vault break-glass (`msp-security-policy.json`)
5. **API integrations & Cursor AI** — Auvik, PSA, RMM, vault, incidents, Cursor troubleshoot addon (`msp-engineer-settings.json` snapshot)
6. **Engineer standalone package** — branded folder with `YourOrg-Engineer-Install.exe`

The engineer package contains only `pathfinder.exe` and `pfseed.exe` in `bundle\` — not master setup or API admin exes.

## Engineer standalone install (`pfengineer-install.exe` / branded copy)

Distribute the folder from step 6 above. Engineers:

1. Double-click `YourOrg-Engineer-Install.exe`
2. Click **Install**
3. Open Pathfinder and sign in with their work account

Pre-staged from the package: enrollment, branding, security policy, API settings, Cursor AI. Engineers do **not** run Azure/Google registration or security admin tools. Add or change Auvik and other integrations in **Settings → Tools** after install.

## Standalone MSP setup (`pfsetup-apis.exe`)

No cloud sign-in required. Configure:

- **Integrations** — Auvik, Domotz, PSA, RMM, IT Glue, Hudu, Passportal, PagerDuty, Opsgenie, etc.
- **Cursor AI** — API key and Troubleshoot addon (side pane + Ops agent)

Use after solo install, or when integrations change without re-running full MSP admin setup.

## API integration setup (standalone admin tool)

| Tool | Purpose |
| --- | --- |
| `pfsetup-apis.exe` | Standalone MSP setup: Auvik, PSA, vault, incidents, **Cursor AI** |

Also embedded in the full MSP wizard (step 5). Use standalone when integrations or Cursor change without re-running cloud auth.

Covers: Auvik, Domotz, NinjaOne, Datto, Automate, N-central, IT Glue, Hudu, Passportal, ConnectWise, Autotask, Halo, PagerDuty, Opsgenie.

## Bundled admin layout

`%LOCALAPPDATA%\PathfinderSSH-MSP\bin\`

| Binary | Role |
| --- | --- |
| `pathfinder.exe` | Main app |
| `pfseed.exe` | Headless sync / seeds |
| `pfinstall.exe` | IT file installer |
| `pfsetup-o365.exe` | Full MSP setup (Microsoft 365) |
| `pfsetup-google.exe` | Full MSP setup (Google Workspace) |
| `pfsetup-apis.exe` | Standalone MSP setup (integrations + Cursor AI) |
| `pfengineer-install.exe` | Template for engineer packages (admin use only) |

## Build

```powershell
.\build-windows.ps1 -Targets installers
```

Builds: pathfinder, pfseed, pfinstall, pfengineer-install, pfsetup-o365, pfsetup-google, pfsetup-apis.

## User data

| Path | Contents |
| --- | --- |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\` | Install root, `msp-enrollment.json`, branding, security policy |
| `%USERPROFILE%\.pathfinderssh\` | sessions, vault, maps, `settings.json` |

Full auth detail: [AUTH.md](AUTH.md).
