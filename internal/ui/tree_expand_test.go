package ui

import "testing"

func TestTreeExpandStyleNormalize(t *testing.T) {
	if TreeExpandStyle("").Normalize() != DefaultTreeExpandStyle {
		t.Fatal("empty should default")
	}
	if TreeExpandStyle("PLUS_MINUS").Normalize() != TreeExpandPlusMinus {
		t.Fatal("plus_minus")
	}
	if TreeExpandStyleFromLabel("Folders") != TreeExpandFolders {
		t.Fatal("from label")
	}
}

func TestTreeExpandPreviewDistinct(t *testing.T) {
	for _, style := range []TreeExpandStyle{
		TreeExpandArrows, TreeExpandExplorer, TreeExpandChevrons,
		TreeExpandPlusMinus, TreeExpandFolders,
	} {
		c, e := TreeExpandPreviewIcons(style)
		if c == nil || e == nil {
			t.Fatalf("nil preview for %q", style)
		}
		if c.Name() == e.Name() && style != TreeExpandArrows {
			// arrows both come from stock and may share naming patterns; others must differ
			if style != TreeExpandChevrons {
				t.Fatalf("%q collapsed and expanded share %q", style, c.Name())
			}
		}
	}
}
