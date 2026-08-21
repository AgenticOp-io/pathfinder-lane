# Pathfinder — Frontend Polish & SecureCRT-Replacement Roadmap

**Status (AgenticOp track):** Phase A partial · Phase B SecureCRT import **shipped** · Phase C button bar + SFTP library **shipped** · GUI rebuild on AgenticOps products  

Work lives on branch `feature/crt-frontend-roadmap` → https://github.com/AgenticOp-io/pathfinderssh  

---

## Done in this pass

| Item | Where |
| --- | --- |
| SecureCRT importer (no passwords) | `internal/crtimport`, `pfseed import-securecrt` |
| CSV import (VanDyke headers) | `pfseed import-csv` |
| Path-encoded deep folders (`A / B / C`) | CRT import → `sessions.yaml` folders |
| Folder session counts in tree | `internal/ui/treemodel.go` |
| Recent-session touch on activate | `internal/recent` + host |
| Button bar (`buttons.yaml`, active/all) | `internal/buttons` + shell SendTo* |
| SFTP file transfer UI | Session → Transfer files / tab Files action |
| YAML session scripts | `internal/scripts` + Scripts toolbar |

| File → Import SecureCRT | `cmd/pathfinder` |
| First-run SecureCRT import | in-app dialog (`pathfinder.exe`) |
| App install / shortcuts | `pathfinder.exe` / `-install` (no START.bat) |

| Rebuilt `pathfinder.exe` + `pfseed.exe` | `products/pathfinder/windows/` |

Verified dry-run on this machine: **786** importable / **19** skipped / **206** folders from VanDyke Config.

## Still open (continue on same branch)

### Phase A polish
- Recent strip UI in the tree panel (persistence is done)
- Quick Connect compact bar
- Session form progressive disclosure
- Keyboard shortcuts (Ctrl+W / Ctrl+Tab)
- Discovery actions overflow grouping

### Phase B
- True nested folders (B2a) — **done**: sessions.yaml v2, recursive tree UI, CRT import nests Customers → site → vendor
- Re-import merge UX feedback in wizard

### Phase C
- SFTP side-panel applet (library ready)
- Port-forward UI
- Armable send-to-all chat box (button scope=all works)
- Windows SSH agent named pipe

### Phase D
- Crawl from selected folder / tags
- Map click jump-path preservation

---

(Original roadmap body follows for reference.)

