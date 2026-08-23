# MSP feature gaps — engineer augment, not PSA replacement

PathfinderSSH MSP is the **engineer's console**: SSH, crawl, maps, capture, vault, and sync targets/credentials from PSA/RMM/doc platforms. It does **not** aim to replace ConnectWise ticketing, Datto fleet automation, team NOC wallboards, or billing.

This doc tracks **CRT parity**, **MSP-specific engineer gaps**, and **what has shipped** since the initial gap analysis.

## Positioning

| Do in Pathfinder | Keep in PSA / RMM / docs / incident apps |
| --- | --- |
| Dial, crawl, map, capture, macros | Ticket lifecycle, SLAs, billing |
| Sync `Customers/<name>/` from PSA | Full CRM / contract management |
| Import inventory IPs from RMM/monitoring | Fleet patching, remote scripts at scale |
| Vault credentials + session linking | Authoritative password policy store |
| Bind incident + document engineer work | On-call routing, escalation policies |
| Change-window ops on customer sessions | Customer comms templates |

## Shipped (engineer-facing)

| Area | Notes |
| --- | --- |
| Customers / Unassigned inventory | MSP filing model vs CRT folder archaeology |
| SecureCRT import wizard | Customer root picker; nest preserved |
| PSA customer sync | ConnectWise, Autotask, Halo, `psa-customers.json` |
| Inventory sync | Auvik (+ periodic/tunnel), Domotz, NinjaOne, Datto RMM, Automate, N-central |
| Credential import | IT Glue, Hudu, Passportal → vault + session link |
| Button bar + YAML macros | Send / WaitFor / WaitRegex |
| SFTP dialog | Live SSH transfer |
| Port forwards | Local / remote / dynamic |
| Tiled / split sessions | Multi-session layout |
| Keyboard shortcuts | Tab/window helpers |
| Session tree filter | Search large estates |
| Script recorder | Record sends into YAML scripts |
| Read-only / change window | Session policy gates |
| **PagerDuty work context (Phase 1)** | Bind incident, status bar, document scrollback as incident note + local evidence zip |
| **Ops desk shell (Phase 2)** | Bind incident → filter tree to customer, per-customer macros, vault scope |
| **Post-change capture pack** | Scrollback + customer map + running-config captures in one zip |
| **Opsgenie adapter** | Same incident bridge as PagerDuty |
| **Send-to-customer** | Bottom send row + `scope: customer` on macros/scripts |
| **Vault customer tags** | `customer:<name>` on doc vault import; break-glass in Settings → Ops |
| **Crawl low-confidence filter** | Low conf button + merge hints in crawl summary |
| **Phase 4 engineer augment** | PSA ticket bind/doc, customer handoff export, console fallback, NOC map view, Python script catalog |
| **Auvik automation** | Tenant map, immediate periodic sync, pfseed sync-auvik, stale prune, tunnel stderr |

## Open — CRT / power-user parity

| Feature | Priority | Effort | Status |
| --- | --- | --- | --- |
| Connect bar (adhoc host without save) | P1 | S | **Shipped** — bottom connect + send row |
| Dependent / chained jump UI polish | P1 | M | **Shipped** — jump_hosts.yaml multi-hop dial |
| Chat / send-to-all armable input | P1 | S | **Shipped** — bottom send row restored |
| Richer script language (Python path) | P1 | L | **Partial** — catalog in `python-scripts/` + run dialog |
| Named firewall / SOCKS profiles | P2 | M | **Partial** — saved port-forward profiles |
| CRT re-import merge report | P1 | M | **Shipped** — per-folder merge dialog |
| Windows OpenSSH agent integration | P1 | M | **Shipped** — OpenSSH agent pipe via `sshcore` |
| Zmodem / classic modem transfers | P3 | L | Open |
| X11 forwarding | P3 | M | Open |
| SecureFX-class dual-pane file client | P2 | XL | Open |

## Open — MSP engineer augment

| Feature | Priority | Effort | Status |
| --- | --- | --- | --- |
| Customer-scoped ops desk shell | P0 | L | **Shipped** — bind incident enters ops desk |
| Per-customer vault scope + break-glass | P0 | L | **Shipped** — customer tags + Settings break-glass |
| Change-window send-to-customer | P1 | M | **Shipped** — send row + scope: customer |
| Post-change capture pack per incident | P1 | M | **Shipped** — capturepack + document dialog |
| Crawl confidence UI + merge suggestions | P1 | M | **Partial** — low-conf filter + merge hints dialog |
| Shared customer map + live overlay (NOC) | P2 | XL | **Partial** — NOC map view opens latest customer map + browser reload |
| Customer export handoff package | P2 | M | **Shipped** — Settings → File tab export zip |
| Serial + SSH dual-path on one node | P2 | M | **Shipped** — `console_fallback` + auto-retry on dial |

## Incident documentation lane

| Feature | Priority | Effort | Status |
| --- | --- | --- | --- |
| PagerDuty bind + document note | P0 | M | **Shipped** — see [PAGERDUTY-API.md](PAGERDUTY-API.md) |
| Opsgenie / Jira Service Management adapter | P1 | M | **Shipped (Opsgenie)** — see [OPSGENIE-API.md](OPSGENIE-API.md) |
| Auto-bind incident from PSA ticket id | P2 | M | **Shipped** — ConnectWise / Autotask / Halo lookup on bind |
| Upload evidence to ticket PSA (not only PD note) | P2 | L | **Shipped** — CW/Autotask/Halo note + attachment upload (tenant-dependent) |

## Explicitly not building (here)

- PSA ticket creation/editing as primary workflow
- RMM remote script runner / patch orchestration
- Team incident wallboards or live session broadcasting to managers
- Billing, contracts, or sales CRM

## Related docs

- [MSP-FEATURES.md](MSP-FEATURES.md) — UI catalog of what exists today
- [MSP-INTEGRATION-STACK.md](MSP-INTEGRATION-STACK.md) — cohesive sync + augment lanes
- [PSA-TICKET-API.md](PSA-TICKET-API.md) — PSA ticket bind + document work
- [ROADMAP-FRONTEND.md](../ROADMAP-FRONTEND.md) — UI roadmap
