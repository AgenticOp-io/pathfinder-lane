// internal/ui/tree_expand.go
// Session-tree branch expand/collapse glyphs (Fyne Tree uses theme icons).
//
// widget.Tree's branch control calls theme.NavigateNextIcon() when closed and
// theme.MoveDownIcon() when open. Those go through theme.Current().Icon(...),
// so NativeTheme.Icon remaps them from Settings.TreeExpandStyle.
package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// TreeExpandStyle is the expand/collapse glyph pair on the session inventory tree.
type TreeExpandStyle string

const (
	TreeExpandArrows    TreeExpandStyle = "arrows"     // Fyne stock → / ↓
	TreeExpandExplorer  TreeExpandStyle = "explorer"   // filled twisties ▶ / ▼
	TreeExpandChevrons  TreeExpandStyle = "chevrons"   // → / ▾ dropdown
	TreeExpandPlusMinus TreeExpandStyle = "plus_minus" // + / −
	TreeExpandFolders   TreeExpandStyle = "folders"    // folder / open folder
)

// DefaultTreeExpandStyle is the Explorer-style twisty (clearest for folder trees).
const DefaultTreeExpandStyle = TreeExpandExplorer

// TreeExpandStyleChoices are labels in Settings → Appearance.
var TreeExpandStyleChoices = []string{
	"Explorer twisties",
	"Arrows (Fyne default)",
	"Chevrons",
	"Plus / minus",
	"Folders",
}

var treeExpandLabelToKey = map[string]TreeExpandStyle{
	"Explorer twisties":    TreeExpandExplorer,
	"Arrows (Fyne default)": TreeExpandArrows,
	"Chevrons":             TreeExpandChevrons,
	"Plus / minus":         TreeExpandPlusMinus,
	"Folders":              TreeExpandFolders,
}

var treeExpandKeyToLabel = map[TreeExpandStyle]string{
	TreeExpandExplorer:  "Explorer twisties",
	TreeExpandArrows:    "Arrows (Fyne default)",
	TreeExpandChevrons:  "Chevrons",
	TreeExpandPlusMinus: "Plus / minus",
	TreeExpandFolders:   "Folders",
}

func (s TreeExpandStyle) Normalize() TreeExpandStyle {
	switch TreeExpandStyle(strings.ToLower(strings.TrimSpace(string(s)))) {
	case TreeExpandArrows, TreeExpandExplorer, TreeExpandChevrons,
		TreeExpandPlusMinus, TreeExpandFolders:
		return TreeExpandStyle(strings.ToLower(strings.TrimSpace(string(s))))
	default:
		return DefaultTreeExpandStyle
	}
}

func (s TreeExpandStyle) Label() string {
	if l, ok := treeExpandKeyToLabel[s.Normalize()]; ok {
		return l
	}
	return treeExpandKeyToLabel[DefaultTreeExpandStyle]
}

// TreeExpandStyleFromLabel maps a settings dropdown label to a key.
func TreeExpandStyleFromLabel(label string) TreeExpandStyle {
	if k, ok := treeExpandLabelToKey[strings.TrimSpace(label)]; ok {
		return k
	}
	return DefaultTreeExpandStyle
}

// treeBranchIcon returns the resource Fyne's Tree should use for name, or nil
// to keep the stock theme icon.
//
// Closed branch → IconNameNavigateNext. Open branch → IconNameMoveDown.
func treeBranchIcon(style TreeExpandStyle, name fyne.ThemeIconName) fyne.Resource {
	style = style.Normalize()
	switch name {
	case theme.IconNameNavigateNext: // collapsed
		switch style {
		case TreeExpandArrows:
			return nil
		case TreeExpandChevrons:
			return theme.DefaultTheme().Icon(theme.IconNameNavigateNext)
		case TreeExpandPlusMinus:
			return theme.DefaultTheme().Icon(theme.IconNameContentAdd)
		case TreeExpandFolders:
			return theme.DefaultTheme().Icon(theme.IconNameFolder)
		default: // explorer
			return themedEmbedIcon("assets/icons/tree-collapsed.svg")
		}
	case theme.IconNameMoveDown: // expanded
		switch style {
		case TreeExpandArrows:
			return nil
		case TreeExpandChevrons:
			return theme.DefaultTheme().Icon(theme.IconNameArrowDropDown)
		case TreeExpandPlusMinus:
			return theme.DefaultTheme().Icon(theme.IconNameContentRemove)
		case TreeExpandFolders:
			return theme.DefaultTheme().Icon(theme.IconNameFolderOpen)
		default: // explorer
			return themedEmbedIcon("assets/icons/tree-expanded.svg")
		}
	}
	return nil
}

// TreeExpandPreviewIcons returns collapsed and expanded glyphs for the settings preview.
func TreeExpandPreviewIcons(style TreeExpandStyle) (collapsed, expanded fyne.Resource) {
	style = style.Normalize()
	if c := treeBranchIcon(style, theme.IconNameNavigateNext); c != nil {
		collapsed = c
	} else {
		collapsed = theme.NavigateNextIcon()
	}
	if e := treeBranchIcon(style, theme.IconNameMoveDown); e != nil {
		expanded = e
	} else {
		expanded = theme.MoveDownIcon()
	}
	return collapsed, expanded
}
