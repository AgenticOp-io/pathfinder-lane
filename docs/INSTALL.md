# Install and update

PathfinderSSH MSP has **two** install paths. API keys (Auvik, PSA, Cursor, …) are **never** collected in install wizards — they live in **Settings → Tools** after Pathfinder is installed.

| Who | Tool | What it does |
| --- | --- | --- |
| **Anyone (standalone)** | `pfinstall.exe` | Installs the app for this Windows profile. No API or cloud setup in the wizard. |
| **MSP admin** | `pfsetup-msp.exe` | **One** MSP wizard: branding → cloud auth (M365 or Google) → security → engineer laptop packages. |
| **Engineer** | `*-Engineer-Install.exe` | Branded package from the MSP wizard. Sign in with work account; APIs in Settings if needed. |

## 1. Standalone installation (`pfinstall.exe`)

| Action | Command |
| --- | --- |
| Graphical install | Double-click `pfinstall.exe` or `pfinstall.exe -install-gui` |
| Silent install | `pfinstall.exe -install` |
| Update binaries | `pfinstall.exe -update` |
| Uninstall | `pfinstall.exe -uninstall` |

Flow: **Ready → Install → Complete → Open Pathfinder**.

**Local credentials (no IT Glue required):**
- Type username and password on each session when you connect, or
- **Settings → File → Manage credentials** to store them in the encrypted local vault
- Optional default SSH user / vault credential name: **Settings → Tools**

Optional integrations (Auvik, PSA, Cursor, …) also live in **Settings → Tools**.

## 2. MSP installation (`pfsetup-msp.exe`)

Run once per tenant (super admin). Steps:

1. **Cloud provider** — Microsoft 365 or Google Workspace  
2. **Branding** — org name, product title, logo  
3. **Cloud authentication** — app registration + verify tenant sign-in  
4. **Security policy** — read-only mode, change windows, vault break-glass  
5. **Engineer packages** — create branded `YourOrg-Engineer-Install.exe` folders  

APIs are **not** in this wizard. Optionally set them in **Settings → Tools** on the admin PC before building an engineer package so credentials ship with the pack; otherwise engineers add them in Settings after install.

Shortcuts that lock the provider (same wizard): `pfsetup-o365.exe`, `pfsetup-google.exe`.

## Engineer package

Distribute the folder from MSP step 5. Engineers:

1. Run `YourOrg-Engineer-Install.exe`  
2. Install  
3. Open Pathfinder and sign in  

No Azure/Google registration or security admin on engineer PCs.

## Bundled layout

`%LOCALAPPDATA%\PathfinderSSH-MSP\bin\`

| Binary | Role |
| --- | --- |
| `pathfinder.exe` | Main app |
| `pfseed.exe` | Headless sync / seeds |
| `pfinstall.exe` | Standalone installer |
| `pfsetup-msp.exe` | **MSP setup** (the one admin wizard) |
| `pfsetup-o365.exe` / `pfsetup-google.exe` | Same wizard with provider pre-selected |
| `pfengineer-install.exe` | Template for engineer packages |
| `pfsetup-apis.exe` | Deprecated stub → use Settings → Tools |

## Build

```powershell
.\build-windows.ps1 -Targets installers
```

## User data

| Path | Contents |
| --- | --- |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\` | Install root, enrollment, branding, security policy |
| `%USERPROFILE%\.pathfinderssh\` | sessions, vault, maps, `settings.json` |

Auth detail: [AUTH.md](AUTH.md).
