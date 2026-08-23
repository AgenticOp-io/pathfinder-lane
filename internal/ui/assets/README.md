# Product logo / icon assets

Default mark: **AgenticOps** (`agenticops-logo.svg` → PNG).

| File | Use |
|------|-----|
| `app-logo.png` | About dialog / splash (embedded default) |
| `app-icon.png` | Fyne window icon (embedded) |
| `pathfinderlogo.png` | Legacy About filename (same image) |
| `agenticops-logo.svg` | Source artwork |
| `app-icon.ico` | Optional; PE icon uses PNGs under `cmd/pathfinder/winres/` |

## Customize About / window icon (no rebuild)

Drop a PNG at:

- `%USERPROFILE%\.pathfinderssh\logo.png`, or
- `%LOCALAPPDATA%\PathfinderSSH-MSP\logo.png`, or
- set `PATHFINDERSSH_LOGO=C:\path\to\logo.png`, or
- `pathfinder.exe -logo C:\path\to\logo.png`

## Customize the `.exe` icon in Explorer

Replace / regenerate the `icon-*.png` files in `cmd/pathfinder/winres/` (max 256×256), then:

```powershell
go-winres make --in cmd/pathfinder/winres/winres.json --out cmd/pathfinder/rsrc --arch amd64,arm64
```

`build-windows.ps1` can run `go-winres` when the tool is on PATH.
