# Pathfinder — Frontend Polish & SecureCRT-Replacement Roadmap

**Status (AgenticOp track):** Phase A–C P0 CRT gaps + MSP ops scaffolding **shipped** on `feature/crt-frontend-roadmap`  

Work lives on branch `feature/crt-frontend-roadmap` → https://github.com/AgenticOp-io/pathfinderssh-msp  

---

## Done in this pass

| Item | Where |
| --- | --- |
| Persistent button bar (bottom strip + All-tabs arm) | `internal/ui/buttonbar.go` + shell `SetBottom` |
| Quick Connect + armable Send chat (active / all / customer) | `internal/ui/opsstrip.go` |
| Keyboard shortcuts Ctrl+W / Ctrl+Tab / Ctrl+Shift+Tab / Ctrl+L | `installShortcuts` |
| Session tree filter focus (Ctrl+L) | `SessionTree.FocusFilter` |
| SFTP as DocTabs applet | `KindSFTP` + `NewSFTPView` |
| Port-forward UI (local / remote / SOCKS5) | `internal/portfwd` + `portfwddialog.go` |
| Send-to-customer + guarded multi-send | shell `SendToMatching` + Ops menu |
| Per-customer vault tag hint (`customer:Name`) | vault manager + `sessions.CustomerTag` |
| Ticket evidence pack (zip scrollbacks) | `internal/evidence` + Session menu |
| PSA → Customers sync scaffold | `internal/psasync` + Ops menu stub |
| CRT re-import merge feedback | `applyImport` messaging |

| SecureCRT importer / CSV / nested folders / maps / vault | earlier on this branch |

## Still open

- Script recorder + Python scripting
- Tiled multi-session layout
- Windows SSH agent named pipe
- Real PSA adapters (ConnectWise / Autotask / …)
- Read-only / change-window policy engine beyond guarded send
- Crawl confidence scoring UI
- Recent strip UI in the tree panel

---

(Original roadmap body follows for reference.)
