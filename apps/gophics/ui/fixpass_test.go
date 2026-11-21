package ui

import (
	"testing"

	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/shell"

	"gguix/core"
)

// Fix pass 1 regressions (results/gophics/fixpass.md).

// Issue 6: app shortcuts work while a text field holds focus.
func TestFixShortcutInFocusedField(t *testing.T) {
	hs := newHarness(t, 0)
	hs.ui(hs.a.FocusSearch) // Autofocus the search field
	hs.h.Type("ab")
	hs.h.Render()
	if hs.a.Search != "ab" {
		t.Fatalf("search field not focused: %q", hs.a.Search)
	}
	hs.h.KeyMod(shell.Key2, shell.ModSuper)
	hs.h.Render()
	hs.h.Render()
	if hs.a.View != core.ViewCards {
		t.Fatal("Cmd+2 ignored while the search field had focus")
	}
	// Plain Cmd+A stays with the field.
	if isShortcut(shell.Key{Kind: shell.KeyPress, Code: shell.KeyA, Mods: shell.ModSuper}) {
		t.Fatal("Cmd+A must not be an app shortcut")
	}
}

// Issue 2: the vertical list fits the visible width and the horizontal strip
// scrolls header and rows.
func TestFixTableScrollbars(t *testing.T) {
	hs := newHarness(t, 0)
	if hs.a.hCtl.MaxOffset() <= 0 {
		t.Fatal("horizontal strip not scrollable")
	}
	y := hs.a.listTop + hs.a.viewportH + hBarH/2
	hs.h.ScrollAt(geom.Pt{X: 300, Y: y}, geom.Pt{X: -120, Y: -120})
	hs.h.Render()
	hs.h.Render()
	if hs.a.hOff != 120 {
		t.Fatalf("hOff = %v, want 120", hs.a.hOff)
	}
}

// Issue 7: the export name prompt cleans the typed name.
func TestFixExportFileName(t *testing.T) {
	for in, want := range map[string]string{
		"gguix-export.csv": "gguix-export.csv",
		" stock ":          "stock.csv",
		"a/b/c.CSV":        "c.CSV",
		"":                 "",
	} {
		if got := exportFileName(in); got != want {
			t.Errorf("exportFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

// Fix pass 2: the header scrolls with the strip and is clipped to the pane.
func TestFixHeaderClipped(t *testing.T) {
	hs := newHarness(t, 0)
	pane := hs.a.split * 1280
	hs.a.hCtl.JumpTo(hs.a.hCtl.MaxOffset())
	for i := 0; i < 4; i++ {
		hs.h.Render()
	}
	if hs.a.hdrCtl.Offset() != hs.a.hCtl.Offset() {
		t.Fatalf("header offset %v, strip %v", hs.a.hdrCtl.Offset(), hs.a.hCtl.Offset())
	}
	for _, n := range hs.h.Semantics() {
		if n.Label == "Featured" && n.Rect.Max.Y < 100 && n.Rect.Max.X > pane+1 {
			t.Fatalf("header label past the pane: %v (pane %v)", n.Rect, pane)
		}
	}
}
