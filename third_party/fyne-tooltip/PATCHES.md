# Patches (PathfinderSSH MSP)

Upstream: [dweymouth/fyne-tooltip](https://github.com/dweymouth/fyne-tooltip) v0.4.0

## overlay tip layer

`internal/tooltip_layer.go` — when the canvas has a top overlay (menu `OverlayContainer`, dialog `*widget.PopUp`) without `AddPopUpToolTipLayer`, skip showing the tip **without** `fyne.LogError("no tool tip layer created for current overlay")`.

Fyne 2.6 menus are not `*widget.PopUp`, so the library’s popup tip API cannot cover them; dialogs also rarely register a tip layer. Logging on every hover during progress dialogs flooded the console.
