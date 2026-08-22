# Pathfinder — Frontend Polish & SecureCRT-Replacement Roadmap

**Status (AgenticOp track):** CRT P0 + MSP ops gaps largely **shipped** on `feature/crt-frontend-roadmap`  

Work: https://github.com/AgenticOp-io/pathfinderssh-msp  

---

## Done

| Item | Where |
| --- | --- |
| Persistent button bar + All-tabs arm | `buttonbar.go` + `SetBottom` |
| Quick Connect + armable Send chat | `opsstrip.go` |
| Shortcuts Ctrl+W/Tab/L | `installShortcuts` |
| SFTP DocTabs applet | `KindSFTP` + `NewSFTPView` |
| Port forwards L/R/SOCKS5 | `internal/portfwd` |
| Send-to-customer + guarded multi-send | shell + Ops menu |
| Ticket evidence pack | `internal/evidence` |
| Script recorder → YAML | `scripts.Recorder` + Session menu |
| Python session scripts (`crt.Screen`) | `internal/pyrun` |
| Tile / untile terminals | `Shell.ToggleTileLayout` |
| Windows OpenSSH agent named pipe | `sshcore/agent_windows.go` + go-winio |
| Read-only + change window | Settings → Ops + `internal/policy` |
| Crawl confidence column + map field | `crawlrun.Confidence`, `topo.NodeDetails` |
| PSA JSON file sync | `psasync.FileSource` + `psa-customers.json` |
| Cursor account / Cloud Agents API | `internal/cursorapi` |
| Troubleshoot addon (gated) | Settings → Ops enable; Ops / toolbar → Troubleshoot agent modal |
| IT Glue password import + session link | `internal/itglue` + Settings → Tools |
| GUI install wizard (Solo / M365 / Google) | `Install.ps1 -install-gui`, `cmd/pfinstall` |
| OIDC sign-in (Entra + Google) | `internal/idp`, `mspauth`, `mspenroll` |

## Still open / polish

- Live ConnectWise / Autotask / Halo SDKs (JSON file adapter is the shipped integration point)
- Auvik: optional AuvikTunnel CLI bridge for collector-only sites (no REST terminal API)
- Cursor Admin / Analytics enterprise endpoints (models list + Cloud Agents shipped)

## Auvik (inventory sync)

| Capability | Status |
| --- | --- |
| API auth + list tenants/devices | `internal/auvik` |
| Import/sync IPs → `Customers/<client>/Auvik/` | Settings → File → Import / Sync |
| Merge existing sessions (Auvik authority for IPs) | `auvik.SyncTenantTree` |
| Periodic sync all clients | Settings → Tools → Periodic Auvik sync |
| AuvikTunnel on unreachable SSH | Settings → Tools → Auto tunnel + `AuvikTunnel` binary |
| Device SSH passwords from Auvik | **Not available** — API has no login-credential export |
| Headless SSH via Auvik API | **Not available** — Pathfinder SSH or local AuvikTunnel |


## Just landed (this pass)

| Item | Where |
| --- | --- |
| Tiled terminal focus ring + title strip | `tilecell.go` + `Shell.activateTile` |
| Ctrl+Tab cycles active tile when tiled | `Shell.cycleTile` |
| Cursor cloud model picker | `cursoraccountdialog` + `ListModels` |

## Previously landed

| Item | Where |
| --- | --- |
| Script recorder wait_for auto-detect | `scripts.Recorder.NoteOutput` + session `SetOnOutputTee` |
| Map confidence labels + border tiers | `mapweb` viewer.js / app.js |
| Troubleshoot script auto-suggest | `scripts.RankNames` + troubleshoot modal Refresh |
| CRT Green / Amber phosphor labels | `themes/crt-green.yaml`, `crt-amber.yaml` |

---
