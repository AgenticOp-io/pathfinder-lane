# Install and update

PathfinderSSH MSP has **two** install paths. API keys (Auvik, PSA, Cursor, …) are **never** collected in install wizards — they live in **Settings → Tools** after Pathfinder is installed.

| Who | Tool | What it does |
| --- | --- | --- |
| **Anyone (standalone)** | `pfinstall.exe` | Installs the app for this Windows profile. No API or cloud setup in the wizard. |
| **SecureCRT add-on** | `pfcrt-install.exe` | Installs/updates Pathfinder CRT Bridge and rewrites SecureCRT sessions. Independent of Pathfinder. |
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

SecureCRT is a **separate product**. Pathfinder does not rewrite VanDyke sessions.

## SecureCRT add-on (`pfcrt-install.exe`)

Standalone installer for **Pathfinder CRT Bridge**. Double-click to install or update.

- Detects `%AppData%\Roaming\VanDyke\Config\Sessions`
- First run: backup + rewrite matching SSH sessions to localhost proxies
- Re-run: **update** binaries, refresh FortiClient / WireGuard / Zscaler and Auvik lists, remap folders (names will not match), and rewrite SecureCRT sessions for mapped folders
- Restarts the background agent so FortiClient auto-connect/switch is live
- See [CRT-BRIDGE.md](CRT-BRIDGE.md)

Build: `.\build-windows.ps1 -Targets crt` → `dist\windows\pfcrt-install.exe`

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
| `pathfinder-msp.exe` | Cursor IDE bridge for PathfinderSSH MSP (stdio → localhost) |
| `pfseed.exe` | Headless sync / seeds |
| `pfinstall.exe` | Standalone installer |
| `pfsetup-msp.exe` | **MSP setup** (the one admin wizard) |
| `pfsetup-o365.exe` / `pfsetup-google.exe` | Same wizard with provider pre-selected |
| `pfengineer-install.exe` | Template for engineer packages |
| `pfsetup-apis.exe` | Deprecated stub → use Settings → Tools |

## Build

```powershell
.\build-windows.ps1 -Targets installers -Install
.\build-windows.ps1 -Targets crt
```

`crt` builds `pathfinder-crt.exe`, `pfcrt-install.exe`, and `pflane.exe` (SecureCRT companion + last-mile CLI). Pathfinder installers do not include them.

Engineers who live in OpenSSH or PuTTY (Windows, Linux, or Mac) should use `pflane create-all` after the folder map — see [PFLANE.md](PFLANE.md).

This builds `pathfinder.exe`, `pathfinder-msp.exe`, and the setup tools into `dist\windows\`, then runs `pfinstall.exe -install`.

## User data

| Path | Contents |
| --- | --- |
| `%LOCALAPPDATA%\PathfinderSSH-MSP\` | Install root, enrollment, branding, security policy |
| `%USERPROFILE%\.pathfinderssh\` | sessions, vault, maps, `settings.json`, `msp-bridge.json` (while app running) |
| `%LOCALAPPDATA%\PathfinderCRT-Bridge\` | Standalone SecureCRT companion binaries |
| `%USERPROFILE%\.pathfinder-crt\` | CRT companion config, backups, logs |

See also [CURSOR-MSP.md](./CURSOR-MSP.md) for Cursor IDE integration.

Auth detail: [AUTH.md](AUTH.md).
