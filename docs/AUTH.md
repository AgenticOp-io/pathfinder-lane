# Sign-in and access modes

## Modes

| Mode | Use case | Cloud login |
| --- | --- | --- |
| **Solo** | Small shop, lab, air-gapped NOC | None — Windows user + vault password |
| **Microsoft 365** | Entra ID, Conditional Access, Intune | OIDC + PKCE (system browser) |
| **Google** | Google Workspace | OIDC + PKCE (system browser) |

Configure at install (`pfinstall.exe`, `-install-gui`) or later:

```powershell
pathfinder -setup solo
pathfinder -setup o365
pathfinder -setup google
```

## Solo

- No cloud identity in the app
- Audit context: Windows username + machine name
- Vault master password required for credentials
- Recommended: BitLocker, standard user account, AppLocker allow-list for `pathfinder.exe`

## Microsoft 365 / Entra

- Authorization code + PKCE in system browser
- Tenant lock via enrollment config (prevents personal Microsoft accounts)
- Refresh token in Windows Credential Manager
- Re-auth required when token revoked — app does not silently fall back to anonymous

IT prerequisites: Entra app registration, redirect URI `http://127.0.0.1:<port>/callback` or custom scheme. Optional script: `products/pathfinder-msp/deploy/entra/`.

## Google Workspace

- Same OIDC pattern as Entra
- Workspace admin configures OAuth client + authorized domains

## Packages

| Package | Role |
| --- | --- |
| `internal/idp/` | OIDC provider abstraction |
| `internal/mspenroll/` | Enrollment / tenant lock JSON |
| `internal/mspauth/` | Token storage, login UI glue |

## Enrollment artifacts

| Artifact | Path |
| --- | --- |
| Org enrollment | `%LOCALAPPDATA%\PathfinderSSH-MSP\msp-enrollment.json` |
| Engineer session | `~/.pathfinderssh/auth-session.json` + refresh token in OS keyring |
| Vault | `~/.pathfinderssh/vault.enc` (all modes) |

CLI: `pathfinder --enroll` (optional `-install` to copy binary then enroll).

## Roadmap (not shipped)

- Per-engineer **customer folder scoping** (Tier 1 vs senior)
- Entra **app roles** / group-based RBAC in the session tree
- Live PSA role sync as RBAC source of truth
- Offline grace window policy (IT-configurable)
- Credential broker / control plane (vs bring-your-own IdP only)

Production hardening: public client + PKCE only, tenant/domain lock, fail-closed on revoked tokens, DPAPI / Credential Manager for refresh tokens — never plaintext in `settings.json`.
