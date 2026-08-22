# MSP installation & access control (plan)

Planning doc for a **first-run / installation routine** that offers three paths:

1. **Local** — standalone install (today’s behavior, hardened)
2. **Microsoft 365 / Entra ID** — work-account sign-in
3. **Google Workspace** — work-account sign-in

This is **not implemented yet**. It describes options, security trade-offs, and a recommended phased build for PathfinderSSH MSP (`pathfinderssh-msp`).

**Related today:** `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pathfinder.exe`, encrypted vault, Auvik sync, auto-tunnel. **No OIDC/OAuth in the app today.**

---

## Goals

| Goal | Why it matters for MSPs |
| --- | --- |
| Know **who** opened the app | Audit, offboarding, SOC questions |
| **Revoke** access without visiting every laptop | Engineer left → disable account |
| **MFA / device compliance** at login | Stolen laptop ≠ instant estate access |
| Optional: **limit which customers** an engineer sees | Tier 1 vs senior, subcontractor scope |
| Keep **GPL desktop app** + local vault model | No forced cloud lock-in for small shops |

---

## Installation routine (proposed UX)

### When it runs

| Trigger | Action |
| --- | --- |
| First launch after `Update-Install.ps1` / MSI | Full **Setup wizard** |
| Re-run from Settings → **Account & access** | Change mode (with admin policy guard) |
| IT silent deploy | Skip wizard via registry/JSON policy (force Local or force Entra tenant) |

### Wizard steps (all paths)

1. **Welcome** — MSP edition, GPL notice, link to SECURITY stance
2. **Access mode** (required) — pick one card:
   - **Local**
   - **Microsoft work account** (Entra ID / M365)
   - **Google work account** (Workspace)
3. **Path-specific** (see below)
4. **Vault** — create or unlock credential vault (all paths; secrets stay local unless broker option chosen later)
5. **Optional integrations** — Auvik API, PSA customer import, AuvikTunnel path
6. **Done** — shortcuts, default folders `Customers/` / `Unassigned/`

### Path: Local install

**Best for:** small MSP, air-gapped lab, break-glass NOC PC, pilots.

| Step | Behavior |
| --- | --- |
| Identity | No cloud login. Operator is whoever runs Windows (implicit OS session). |
| Vault | Required — master password + optional OS keyring “Remember” (existing). |
| Policy file (optional) | `access-policy.json` beside settings — IT can pre-seed allowed features. |
| Hardening prompts | Recommend BitLocker, standard user (not admin), Intune/AppLocker allow-list for `pathfinder.exe`. |

**Security note:** Local mode does **not** prove which human is using the app. Audit lines should include **Windows username + machine name** until OIDC is added.

### Path: Microsoft 365 / Entra ID

**Best for:** MSPs already on Entra, Conditional Access, Intune, M365 E3/E5.

| Step | Behavior |
| --- | --- |
| Sign-in | OIDC **authorization code + PKCE** in system browser (or embedded WebView2) |
| Tenant lock | Wizard asks for **tenant ID or verified domain** (or IT preconfigures via policy) so personal Microsoft accounts cannot satisfy login |
| Token storage | Refresh token in **Windows Credential Manager** (or DPAPI-protected file); access token short-lived in memory |
| Post-login | Show signed-in **UPN**, tenant, optional **app roles** / group summary |
| Re-auth | On token expiry / revoke → blocking re-login screen (app does not fall back to anonymous) |

**IT prerequisites (admin):**

- Entra **App registration** (multi-tenant or single-tenant)
- Redirect URI: `http://127.0.0.1:<port>/callback` or custom scheme `pathfinder-msp://auth`
- Optional: **app roles** (e.g. `Pathfinder.User`, `Pathfinder.Admin`, `Pathfinder.Tier1`)
- **Conditional Access** policies targeting this app registration
- Optional: **Group-based** assignment to enterprise app

### Path: Google Workspace

**Best for:** MSPs on Google Workspace (less common for MSP tooling, but valid).

| Step | Behavior |
| --- | --- |
| Sign-in | OIDC + PKCE; **internal** OAuth client if all users in one Workspace |
| Domain lock | Restrict `hd` (hosted domain) claim or admin-configured allowed domains |
| Token storage | Same as Entra — OS credential store |
| Admin | Workspace admin creates OAuth client; can restrict to org only |

**IT prerequisites:**

- Google Cloud project + OAuth client (Desktop or Web + loopback redirect)
- Workspace admin: trust internal app, enforce 2-Step Verification
- Optional: **Context-Aware Access** (Google Beyond Corp / CAA) for device/IP posture

---

## What each IdP can actually control

### Controls that work well with a **desktop** app (Entra & Google)

| Control | Entra ID / M365 | Google Workspace | Applies to Pathfinder |
| --- | --- | --- | --- |
| **MFA at login** | Strong (CA: require MFA) | Strong (2SV + CAA) | **Yes** — gate app launch |
| **Block legacy auth** | Tenant policy | Workspace policy | Indirect — OIDC only |
| **Disable user** | Immediate on next refresh | Immediate | **Yes** — no new tokens |
| **Session lifetime** | CA sign-in frequency / token lifetime | Session length policies | **Yes** — force re-login |
| **Compliant / managed device** | Intune compliance in CA | Context-Aware Access | **Yes** — if CA/CAA configured |
| **Named locations / IP** | CA trusted locations | CAA IP regions | **Yes** — office/VPN only |
| **Sign-in audit** | Entra sign-in logs | Admin audit logs | **Yes** — who authenticated to app |
| **App assignment** | Assign enterprise app to groups | Internal app + groups | **Yes** — only assigned users get tokens |
| **Custom roles in token** | App roles → `roles` claim | Group membership → custom claim | **Yes** — RBAC in app |
| **Guest / B2B** | Entra B2B guests | Google guest users | **Careful** — scope guests narrowly |

### Controls that **do not** come free from IdP login

| Need | IdP alone? | What you still need |
| --- | --- | --- |
| **Which customer folders** user sees | Partial | Map **groups → Customers/** in app or broker |
| **Which devices** user can SSH to | No | App RBAC + vault credential sets |
| **SSH password / key** protection | No | Vault remains local; IdP gates *app* not *device* |
| **Auvik API / tunnel** permission | No | Separate Auvik user + API key |
| **Proof engineer ran command X on device Y** | Partial | Session transcripts + tie log to **OIDC `sub` / UPN** |
| **Central revoke of in-flight SSH** | No | Broker or agent; IdP only stops *next* app open |

### Entra vs Google — security posture (honest comparison)

| Dimension | Entra ID (M365) | Google Workspace | Typical MSP fit |
| --- | --- | --- | --- |
| **Conditional Access maturity** | Very mature (device, app, risk-based) | Good with CAA (often extra SKU) | **Entra wins** for MSPs on M365 |
| **MDM integration** | Intune native | Google endpoint / third-party | **Entra** if already Intune |
| **Desktop OAuth + PKCE** | Well documented | Well documented | Tie |
| **B2B contractors** | Entra B2B common | Less common | **Entra** for MSP vendor access |
| **Audit & SIEM** | Sentinel, Log Analytics | Chronicle / BigQuery export | Depends on stack |
| **Air-gapped / no cloud IdP** | — | — | **Local** only |

**Recommendation for most MSPs on Microsoft stack:** **Entra ID first**, Google as second provider using the same OIDC abstraction in code.

---

## Security options (pick a tier)

### Option A — Local hardened (status quo + polish)

**What:** Local install wizard only; no cloud login.

| Pros | Cons |
| --- | --- |
| Ships fastest; no cloud dependency | No central revoke; weak person-level audit |
| Works offline | Stolen laptop + vault password = full access |
| Fits GPL offline ethos | RBAC is manual (folder discipline) |

**Hardening bundle:** Intune required app, BitLocker, vault mandatory, no “Remember vault” on shared PCs, transcript logging with Windows user.

**Best when:** &lt; 10 engineers, trusted office LAN, pilot phase.

---

### Option B — IdP gate only (“sign in to open”)

**What:** Entra or Google OIDC at startup. Vault and sessions still 100% local.

| Pros | Cons |
| --- | --- |
| MFA + CA via IdP | Everyone still sees all Customers once inside |
| Disable account → app dead on refresh | Tokens on endpoint (protect with DPAPI) |
| Clear audit: UPN in app logs | Does not limit SSH credentials by role |
| Moderate dev effort | IT must register OAuth app |

**Security grade:** **Good** for “only our staff runs Pathfinder,” **not** for least-privilege per customer.

**Best when:** MSP wants offboarding and MFA without building a cloud backend.

---

### Option C — IdP gate + group-based inventory RBAC

**What:** Option B + map IdP groups to allowed **customer roots** (and optional feature flags).

Example Entra setup:

| Group | Pathfinder effect |
| --- | --- |
| `PF-All-Customers` | Full `Customers/*` tree |
| `PF-Tier1` | Only `Customers/{A,B,C}` |
| `PF-Project-X` | Only `Customers/Acme` |

Implementation choices:

1. **Claims in token** — app roles or groups (watch token size; group overage in Entra)
2. **Local policy file** — IT deploys `access-policy.json` per team (signed optional)
3. **Lightweight config pull** — HTTPS JSON from your static host / GitHub after login

| Pros | Cons |
| --- | --- |
| Least privilege inside app | Policy sync and support burden |
| Tier 1 cannot open wrong customer tab | Misconfigured group = ticket storm |
| Still no SSH broker needed | Vault creds must align with RBAC |

**Security grade:** **Strong** for MSP operational separation.

**Best when:** 15+ engineers, mixed tiers, compliance asks “who can touch customer X.”

---

### Option D — Identity broker / control plane (maximum MSP security)

**What:** Small **AgenticOps-hosted** (or customer self-hosted) service:

1. User signs in with Entra/Google **to the broker**
2. Broker returns **short-lived session JWT** + scoped policy (customers, features)
3. Optional: broker holds **no secrets** (same as C) OR broker **vends vault unlock keys / ephemeral creds** (high effort)

| Pros | Cons |
| --- | --- |
| Central revoke (kill session) | You operate a service (uptime, GDPR, cost) |
| Uniform audit across all engineers | GPL + SaaS positioning needs care |
| Can integrate PSA, ticketing, approval workflows | Highest build and ops cost |
| Future: just-in-time elevation | |

**Security grade:** **Strongest** for regulated MSPs and multi-region teams.

**Best when:** SOC2-bound MSP, 50+ seats, need central audit and customer-scoped RBAC with approval.

---

### Option E — Hybrid (recommended phased roadmap)

| Phase | Deliverable | Access mode |
| --- | --- | --- |
| **0 (now)** | Local install + vault + Auvik auto-tunnel | Local |
| **1** | Setup wizard + Local path polished; policy file stub | Local |
| **2** | Entra OIDC gate (Option B) | Local **or** Microsoft |
| **3** | Google OIDC (same code path) | + Google |
| **4** | Group → customer RBAC (Option C) | All paths; cloud modes get live policy |
| **5** | Optional broker (Option D) for enterprise SKU | Cloud policy + audit |

This avoids blocking small shops on cloud while giving a clear enterprise upsell.

---

## Recommended defaults (security)

| Topic | Recommendation |
| --- | --- |
| **Default for new MSP pilots** | Local + mandatory vault + transcripts |
| **Default for production MSP (M365)** | **Entra gate (B)** then **group RBAC (C)** |
| **OAuth client type** | Public client + **PKCE** only; no client secret in desktop binary |
| **Redirect** | Loopback `127.0.0.1` with random port or fixed port + firewall rule |
| **Token storage** | Windows Credential Manager / DPAPI; never plaintext refresh in `settings.json` |
| **Tenant/domain lock** | Required for cloud paths — prevent consumer Gmail / personal Microsoft |
| **Fallback** | **Fail closed** — if IdP unreachable and token expired, block UI (with offline grace window IT configures) |
| **Vault** | Still encrypted local file; IdP does not replace vault |
| **Shared NOC PC** | Local mode + **no** keyring remember + Windows fast user switching |
| **Break-glass** | Local policy file `break_glass: true` only on designated machine |

---

## Installation artifacts (per path)

| Artifact | Local | Entra | Google |
| --- | --- | --- | --- |
| Binary | `%LOCALAPPDATA%\PathfinderSSH-MSP\bin\pathfinder.exe` | Same | Same |
| Settings | `~/.pathfinderssh/settings.json` | + `auth_provider`, `tenant_id` | + `hd` domain |
| Secrets | `vault.json` + keyring | + OAuth refresh in credential store | Same |
| IT policy | `access-policy.json` (optional) | + Entra app registration JSON | + OAuth client config |
| Audit | Transcripts + Windows user | + UPN/`sub` in log prefix | + Google `email`/`sub` |

---

## Open decisions (for you)

1. **Single-tenant vs multi-tenant Entra app** — one app for all MSP customers of AgenticOps vs per-MSP app registration?
2. **Google priority** — Phase 3 or defer if 100% M365 MSP base?
3. **RBAC source of truth** — Entra groups only, or PSA (ConnectWise roles) later?
4. **Offline grace** — 0 hours (strict) vs 24h (field engineer friendly)?
5. **Broker** — build AgenticOp control plane vs document “bring your own” only?

---

## Implementation status (hooks shipped)

| Piece | Path |
| --- | --- |
| Org enrollment file | `%LOCALAPPDATA%\PathfinderSSH-MSP\msp-enrollment.json` (or `%PROGRAMDATA%\PathfinderSSH-MSP\` / `PATHFINDER_MSP_ENROLLMENT`) |
| Engineer session | `~/.pathfinderssh/auth-session.json` + refresh token in OS keyring |
| Package | `internal/mspauth` — enroll, OIDC+PKCE login, session validation |
| First-run UI | Super-admin **Enroll** wizard; engineer **Sign in** wizard |
| CLI | `pathfinder --enroll` (with optional `-install` to copy then enroll) |

JWT signature verification is a **hook** (`TokenVerifier`); default parser is for integration testing — production should plug JWKS verification.


## References

- [Microsoft identity platform — desktop app (PKCE)](https://learn.microsoft.com/en-us/entra/identity-platform/v2-oauth2-auth-code-flow)
- [Google OAuth 2.0 for desktop apps](https://developers.google.com/identity/protocols/oauth2/native-app)
- Pathfinder MSP security stance: [`SECURITY.md`](../../SECURITY.md)
- Current install layout: [`README-MSP.md`](../../README-MSP.md)
