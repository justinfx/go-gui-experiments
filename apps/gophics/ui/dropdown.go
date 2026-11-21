package ui

import (
	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"
)

// Select stands in for theme.Dropdown. theme.Dropdown publishes an
// "expanded" accessibility state, and at the pinned commit the macOS bridge
// answers that with -[GoGPUA11yElement setAccessibilityDisclosing:], which
// AppKit on macOS 26 rejects with an uncaught NSInvalidArgumentException,
// aborting the process at the first frame. Select is the same control from
// stock parts: a theme.Button showing the current option, opening
// theme.ShowMenu anchored under it.
type Select struct {
	Options  []string
	Selected int
	OnChange func(int)
}

func (s Select) CreateState() widget.State { return &selectState{} }

type selectState struct {
	widget.StateBase[Select]
	anchor geom.Rect
}

func (st *selectState) Build(ctx widget.Ctx) widget.Widget {
	w := st.W()
	label := ""
	if w.Selected >= 0 && w.Selected < len(w.Options) {
		label = w.Options[w.Selected]
	}
	return widget.LayoutObserver{
		OnLayout: func(r geom.Rect) { st.anchor = r },
		Child: theme.Button{Label: label + "  ▼", OnTap: func() {
			if w.OnChange == nil {
				return
			}
			items := make([]theme.MenuItem, len(w.Options))
			for i, o := range w.Options {
				i := i
				items[i] = theme.MenuItem{Label: o, OnTap: func() { w.OnChange(i) }}
			}
			theme.ShowMenu(ctx, geom.Pt{X: st.anchor.Min.X, Y: st.anchor.Max.Y + 2}, items)
		}},
	}
}
