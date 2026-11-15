package main

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/base/datatable"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	gx "gguix/core"
)

// frameworkVersion is NSGo's pseudo-version as the build recorded it.
func frameworkVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/nsgo-dev/nsgo" {
				return "NSGo " + d.Version
			}
		}
	}
	return "NSGo (commit 2345b2f)"
}

// pt is a point length.
func pt(v float32) core.Length { return core.Pt(core.Points(v)) }

// rootView returns the app's root component. It binds the model to the
// scope, then builds the whole window: toolbar, split area, status bar and
// the dialogs.
func rootView(a *app) func(cx *core.Scope) core.Node {
	return func(cx *core.Scope) core.Node {
		a.bind(cx)
		colors := ui.ColorsOf(cx)
		cx.SetAppearance(appearanceOf(a.themeSel))
		cx.OnQuit(func(context.Context) { a.flushPrefs() })
		a.installPlatform(cx)
		startBench(cx, a)
		a.restoreSelection()

		tbl := &datatable.Ref{}
		cardsRef := &datatable.Ref{}
		searchRef := &core.Ref{}
		a.scrollFn = func(view, row int) {
			if view == 0 {
				tbl.ScrollToRow(row, false)
			} else {
				cardsRef.ScrollToRow(row, false)
			}
		}
		a.focusSearchFn = searchRef.Focus

		left := core.Show{
			When: func() bool { a.rQuery.Watch(); return a.viewMode == 0 },
			Then: func(cx *core.Scope) core.Node { return tableView(cx, a, tbl) },
			Else: func(cx *core.Scope) core.Node { return cardsView(cx, a, cardsRef) },
		}

		main := base.ResizablePanelGroup{
			Axis:  base.Horizontal,
			Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: pt(0)},
			Panels: []base.ResizablePanel{
				{MinSize: 400, TestID: "left-panel", Style: core.Style{MinWidth: pt(0)}, Children: []core.Node{left}},
				{Size: func() core.Points { return 440 }, MinSize: 320, TestID: "right-panel", Children: []core.Node{detailView(cx, a)}},
			},
		}

		return core.View{
			Style: core.Style{FlexGrow: 1, FlexShrink: 1, Direction: core.Column, MinHeight: pt(0)},
			StyleFunc: func() core.Style {
				return core.Style{Background: colors.Background(), Color: colors.Text(), FontSize: 13}
			},
			Children: []core.Node{
				toolbarView(cx, a, searchRef),
				core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: pt(0), Direction: core.Column}, Children: []core.Node{main}},
				statusBar(cx, a),
				dialogs(cx, a),
			},
		}
	}
}

const (
	cardHeight = 96
	cardGap    = 8
	cardPitch  = cardHeight + cardGap
)

// ---- toolbar -----------------------------------------------------------

func toolbarView(cx *core.Scope, a *app, searchRef *core.Ref) core.Node {
	colors := ui.ColorsOf(cx)

	search := ui.Input{
		Ref:         searchRef,
		Value:       func() string { a.rQuery.Watch(); return a.search },
		OnChange:    a.setSearch,
		Placeholder: "Search name or SKU",
		Label:       "Search",
		TestID:      "search",
		Style:       core.Style{Width: pt(280), FlexShrink: 0},
	}

	catItems := []core.Node{}
	for _, c := range gx.AllCategories {
		c := c
		catItems = append(catItems, ui.MenuItem{
			Text: c.String(), NoCloseOnSelect: true, TestID: "cat-" + c.String(),
			Checked:  func() bool { a.rQuery.Watch(); return a.cats[c] },
			OnSelect: func() { a.toggleCategory(c) },
		})
	}
	catItems = append(catItems, ui.MenuSeparator{}, ui.MenuItem{Text: "Clear", TestID: "cat-clear", NoCloseOnSelect: true,
		Disabled: func() bool { a.rQuery.Watch(); return a.catCount() == 0 },
		OnSelect: a.clearCategories})
	catCountMemo := core.NewMemo(cx, func() int { a.rQuery.Watch(); return a.catCount() })
	categoriesDyn := core.Dynamic(func(cx *core.Scope) core.Node {
		n := catCountMemo.Get()
		label := "Categories"
		if n > 0 {
			label = fmt.Sprintf("Categories (%d)", n)
		}
		return ui.Menu{
			TestID:       "categories",
			Open:         func() bool { a.rDlg.Watch(); return a.catsOpen },
			OnOpenChange: func(open bool, _ base.Reason) { a.catsOpen = open; a.rDlg.Bump() },
			Trigger:      ui.Button{Text: label, Variant: ui.ButtonOutline, TestID: "categories-button"},
			Label:        label,
			Children:     catItems,
		}
	})

	statusOpts := []ui.Option[string]{}
	for _, s := range statusChoices {
		statusOpts = append(statusOpts, ui.Option[string]{Value: s, Label: s})
	}
	status := ui.Select[string]{
		Options: statusOpts, TestID: "status-filter", Label: "Status filter",
		Value:    func() string { a.rQuery.Watch(); return a.statusSel },
		OnChange: a.setStatusFilter,
		Style:    core.Style{Width: pt(150), FlexShrink: 0},
	}

	goInput := ui.Input{
		Placeholder: "Go to row", Label: "Go to row", TestID: "goto", KeyboardType: core.KeyboardNumber,
		Value:    func() string { a.rQuery.Watch(); return a.goTo },
		OnChange: func(s string) { a.goTo = s; a.rQuery.Bump() },
		OnSubmit: func(string) { a.doGo() },
		Style:    core.Style{Width: pt(100), FlexShrink: 0},
	}
	goBtn := ui.Button{Text: "Go", TestID: "go", Variant: ui.ButtonSecondary, OnPress: a.doGo}

	// A ToggleGroup of ToggleButtons: the selected one is filled with the
	// accent, which reads far more clearly than Tabs' subtle segmented thumb.
	viewBtn := func(label string, mode int, id string) core.Node {
		return ui.ToggleButton{
			Text: label, TestID: id,
			On:       func() bool { a.rQuery.Watch(); return a.viewMode == mode },
			OnChange: func(bool) { a.setView(mode) },
		}
	}
	views := ui.ToggleGroup{Label: "View", TestID: "view-tabs", Children: []core.Node{
		viewBtn("Table", 0, "view-table"), viewBtn("Cards", 1, "view-cards"),
	}}

	export := ui.Button{
		Text: "Export CSV", TestID: "export", Variant: ui.ButtonSecondary,
		Disabled: func() bool { a.rExp.Watch(); return a.exp.running },
		OnPress:  func() { a.startExport() },
	}

	themeOpts := []ui.Option[string]{}
	for _, s := range themeChoices {
		themeOpts = append(themeOpts, ui.Option[string]{Value: s, Label: s})
	}
	theme := ui.Select[string]{
		Options: themeOpts, TestID: "theme", Label: "Theme",
		Value:    func() string { a.rQuery.Watch(); return a.themeSel },
		OnChange: a.setTheme,
		Style:    core.Style{Width: pt(110), FlexShrink: 0},
	}

	return core.View{
		TestID: "toolbar",
		Style:  core.Style{Direction: core.Row, Wrap: core.Wrap, AlignItems: core.AlignCenter, Gap: 8, Padding: core.Symmetric(8, 12), FlexShrink: 0},
		StyleFunc: func() core.Style {
			return core.Style{Borders: core.Edges{Bottom: 1}, BorderColor: colors.Border()}
		},
		Children: []core.Node{
			search, categoriesDyn, status, goInput, goBtn, views,
			core.View{Style: core.Style{FlexGrow: 1}},
			export, theme,
		},
	}
}

// doGo is the Go button and the go-to field's Enter.
func (a *app) doGo() {
	if n, ok := parseRowNumber(a.goTo); ok {
		a.goToRow(n)
	}
}

// startExport asks for notification permission (a user action just
// happened) and then starts the export flow.
func (a *app) startExport() {
	a.cx.RequestNotificationPermission(func(core.NotificationPermission) {})
	a.beginExport()
}

// ---- table --------------------------------------------------------------

type colDef struct {
	key, title string
	width      float32
	align      datatable.Align
	sort       gx.SortField // "" when the column does not sort
}

var tableCols = []colDef{
	{"id", "ID", 80, datatable.AlignEndColumn, gx.SortID},
	{"sku", "SKU", 110, datatable.AlignStartColumn, gx.SortSKU},
	{"name", "Name", 260, datatable.AlignStartColumn, gx.SortName},
	{"category", "Category", 110, datatable.AlignStartColumn, gx.SortCategory},
	{"price", "Price", 100, datatable.AlignEndColumn, gx.SortPrice},
	{"quantity", "Qty", 80, datatable.AlignEndColumn, gx.SortQuantity},
	{"stock", "Stock", 120, datatable.AlignStartColumn, ""},
	{"status", "Status", 120, datatable.AlignStartColumn, gx.SortStatus},
	{"rating", "Rating", 110, datatable.AlignStartColumn, gx.SortRating},
	{"updated", "Updated", 150, datatable.AlignStartColumn, gx.SortUpdated},
	{"featured", "Featured", 80, datatable.AlignCenterColumn, gx.SortFeatured},
}

// headerTitle is a column's title with the sort arrow when it is the active
// sort.
func (a *app) headerTitle(c colDef) string {
	if c.sort != "" && c.sort == a.sortField {
		if a.sortDesc {
			return c.title + " ▼"
		}
		return c.title + " ▲"
	}
	return c.title
}

// cellText is the text of a plain cell, "…" for a row that is not loaded.
func (a *app) cellText(row, col int) string {
	a.rRows.Watch()
	it, ok := a.pager.Row(row)
	if !ok {
		return "…"
	}
	switch tableCols[col].key {
	case "id":
		return strconv.FormatInt(it.ID, 10)
	case "sku":
		return it.SKU
	case "name":
		return it.Name
	case "category":
		return it.Category.String()
	case "price":
		return gx.FormatPrice(it.PriceCents)
	case "quantity":
		return strconv.Itoa(int(it.Quantity))
	case "updated":
		return gx.FormatUpdated(it.UpdatedAt)
	case "featured":
		if it.Featured {
			return "✓"
		}
		return ""
	}
	return ""
}

// tableStyles is the look of the table's parts. base's DataTable has no
// look of its own; these are the gallery's parts for it, from ui's colors.
func tableStyles(cx *core.Scope) func() datatable.Styles {
	return ui.Themed(cx, func() datatable.Styles {
		colors := ui.ColorsOf(cx).Now()
		return datatable.Styles{
			Header:        core.Style{Background: ui.Alpha(colors.Inset, 0.5), Color: colors.Muted, Borders: core.Edges{Bottom: 1}, BorderColor: colors.Border},
			HeaderCell:    core.Style{Padding: core.Edges{Left: 8, Right: 4}},
			HeaderContent: core.Style{Gap: 4},
			ResizeHandle:  core.Style{Background: colors.Border},
			MoveIndicator: core.Style{Background: colors.Accent},
			Row:           core.Style{Borders: core.Edges{Bottom: 1}, BorderColor: colors.Line},
			RowSelected:   core.Style{Background: ui.Alpha(colors.Accent, 0.16)},
			Cell:          core.Style{Padding: core.Symmetric(0, 8)},
			FixedEdge:     core.Style{Background: colors.Border},
		}
	})
}

func tableView(cx *core.Scope, a *app, tbl *datatable.Ref) core.Node {
	colors := ui.ColorsOf(cx)
	alignOf := func(key string) core.TextAlign {
		for _, c := range tableCols {
			if c.key == key {
				switch c.align {
				case datatable.AlignEndColumn:
					return core.TextAlignEnd
				case datatable.AlignCenterColumn:
					return core.TextAlignCenter
				}
			}
		}
		return core.TextAlignStart
	}

	render := func(cx *core.Scope, cell func() datatable.Cell) core.Node {
		key := core.NewMemo(cx, func() string { return cell().Key })
		it := func() (gx.Item, bool) {
			a.rRows.Watch()
			c := cell()
			if c.Row < 0 {
				return gx.Item{}, false
			}
			return a.pager.Row(c.Row)
		}
		loaded := core.NewMemo(cx, func() bool { _, ok := it(); return ok })
		placeholder := func(*core.Scope) core.Node { return core.Text{Content: "…", MaxLines: 1} }
		return core.Dynamic(func(cx *core.Scope) core.Node {
			switch k := key.Get(); k {
			case "stock":
				return core.Show{When: loaded.Get, Else: placeholder,
					Then: func(cx *core.Scope) core.Node { return stockBar(cx, it, false, true, "cell-stock") }}
			case "status":
				return core.Show{When: loaded.Get, Else: placeholder,
					Then: func(cx *core.Scope) core.Node { return statusBadge(cx, it, "cell-status") }}
			case "rating":
				return core.Show{When: loaded.Get, Else: placeholder,
					Then: func(cx *core.Scope) core.Node { return stars(cx, it, "cell-stars") }}
			default:
				return core.Text{
					ContentFunc: func() string { c := cell(); return a.cellText(c.Row, colIndex(c.Key)) },
					MaxLines:    1,
					Style:       core.Style{Width: core.Pct(100), MinWidth: pt(0), TextAlign: alignOf(k)},
				}
			}
		})
	}

	renderHeader := func(cx *core.Scope, cell func() datatable.HeaderCell) core.Node {
		def := func() colDef { return tableCols[cell().Column] }
		return core.View{
			Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: pt(0)},
			OnPress: func(core.PressEvent) {
				if c := def(); c.sort != "" {
					a.setSort(c.sort)
				}
			},
			Children: []core.Node{core.Text{
				MaxLines:    1,
				ContentFunc: func() string { a.rQuery.Watch(); return a.headerTitle(def()) },
				StyleFunc: func() core.Style {
					return core.Style{TextAlign: alignOf(def().key), FontWeight: core.FontWeightMedium}
				},
			}},
		}
	}

	columns := func() []datatable.Column {
		out := make([]datatable.Column, len(tableCols))
		for i, c := range tableCols {
			out[i] = datatable.Column{Key: c.key, Title: c.title, Width: core.Points(c.width), Align: c.align, MinWidth: 40}
		}
		return out
	}

	return core.View{
		TestID: "table-box",
		Style:  core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: pt(0), MinWidth: pt(0)},
		Children: []core.Node{datatable.DataTable{
			Ref: tbl, TestID: "table", Label: "Items",
			Columns:          columns,
			RowCount:         func() int { a.rRows.Watch(); return max(a.pager.Total(), 0) },
			Text:             a.cellText,
			NoRowHeader:      true,
			NoColumnSelect:   true,
			NoLoopSelection:  true,
			RenderCell:       render,
			RenderHeaderCell: renderHeader,
			Selection: func() datatable.Selection {
				a.rSel.Watch()
				if a.selRow < 0 {
					return datatable.NoTableSelection
				}
				return datatable.Selection{Row: a.selRow, Column: -1}
			},
			Empty:      core.Text{Content: "No items match.", TestID: "empty"},
			StylesFunc: tableStyles(cx),
			StyleFunc: func() core.Style {
				return core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: pt(0), BorderWidth: 1, BorderColor: colors.Border(),
					Background: colors.Background(), Color: colors.Text()}
			},
			OnSelectionChange: func(s datatable.Selection) {
				if s.Row >= 0 && s.Row != a.selRow {
					a.requestSelect(s.Row)
				}
				// Keep the table's selection what the model says (the dirty
				// guard may have refused the change).
				a.rSel.Bump()
			},
			OnVisibleRowsChange: func(start, end int) {
				if a.viewMode == 0 {
					a.setVisible(start, end-1)
				}
			},
		}},
	}
}

func colIndex(key string) int {
	for i, c := range tableCols {
		if c.key == key {
			return i
		}
	}
	return 0
}

// ---- cards ---------------------------------------------------------------

// cardsView is the Cards view: one virtualized column of custom cards. It is
// a base DataTable with a single column and a 104 pt row (96 pt card + 8 pt
// gap), not base.List, because only the DataTable scales its scroll range
// past a few million points. core.List lays out in 32-bit floats and is
// not exact past 2^24 points, which 1,000,000 cards of 104 pt (104 million)
// are. The one-point header is invisible.
func cardsView(cx *core.Scope, a *app, tbl *datatable.Ref) core.Node {
	colors := ui.ColorsOf(cx)
	width := core.NewSignal(cx, core.Points(600))

	render := func(cx *core.Scope, cell func() datatable.Cell) core.Node {
		idx := func() int { return cell().Row }
		it := func() (gx.Item, bool) {
			a.rRows.Watch()
			if idx() < 0 {
				return gx.Item{}, false
			}
			return a.pager.Row(idx())
		}
		loaded := core.NewMemo(cx, func() bool { _, ok := it(); return ok })
		isSel := func() bool { a.rSel.Watch(); return a.selRow == idx() }
		frame := func(children ...core.Node) core.Node {
			return core.View{
				TestID: "card",
				LabelFunc: func() string {
					if i, ok := it(); ok {
						return i.Name
					}
					return "Loading…"
				},
				Selected: isSel,
				StyleFunc: func() core.Style {
					bw, bc := core.Points(1), colors.Border()
					if isSel() {
						bw, bc = 2, colors.Accent()
					}
					return core.Style{Width: core.Pct(100), Height: pt(cardHeight), Padding: core.All(12 - (bw - 1)), BorderWidth: bw, BorderColor: bc,
						CornerRadius: 10, Background: colors.Surface(), Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12, Overflow: core.Hidden}
				},
				Children: children,
			}
		}
		return core.Show{
			When: loaded.Get,
			Else: func(*core.Scope) core.Node {
				return frame(core.Text{Content: "Loading…", Style: core.Style{FontWeight: core.FontWeightBold}})
			},
			Then: func(cx *core.Scope) core.Node {
				muted := ui.Colored(colors.Muted)
				mid := core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: pt(0), Gap: 2}, Children: []core.Node{
					core.Text{ContentFunc: func() string { i, _ := it(); return i.Name }, MaxLines: 1, TestID: "card-name",
						Style: core.Style{FontWeight: core.FontWeightBold}},
					core.Text{ContentFunc: func() string { i, _ := it(); return i.SKU + " · " + i.Category.String() }, MaxLines: 1,
						Style: core.Style{FontSize: 12}, StyleFunc: muted},
					core.Text{ContentFunc: func() string { i, _ := it(); return gx.NotesPreview(i.Notes) }, MaxLines: 1,
						EllipsizeMode: core.EllipsizeTail, Style: core.Style{FontSize: 12}, StyleFunc: muted},
					core.View{Style: core.Style{Direction: core.Row, Gap: 6, AlignItems: core.AlignCenter}, Children: []core.Node{
						statusBadge(cx, it, "card-status"),
						core.Show{When: func() bool { i, _ := it(); return i.Featured }, Then: func(cx *core.Scope) core.Node { return featuredChip(cx) }},
					}},
				}}
				right := core.View{Style: core.Style{Width: pt(160), FlexShrink: 0, AlignItems: core.AlignEnd, Gap: 4}, Children: []core.Node{
					core.Text{ContentFunc: func() string { i, _ := it(); return gx.FormatPrice(i.PriceCents) }, TestID: "card-price",
						Style: core.Style{FontSize: 17, FontWeight: core.FontWeightBold}},
					stars(cx, it, "card-stars"),
					stockBar(cx, it, true, false, "card-stock"),
				}}
				return frame(categoryTile(cx, it), mid, right)
			},
		}
	}

	return core.View{
		TestID: "cards-box",
		Style:  core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: pt(0), MinWidth: pt(0)},
		Children: []core.Node{datatable.DataTable{
			Ref: tbl, TestID: "cards", Label: "Cards",
			Columns: func() []datatable.Column {
				return []datatable.Column{{Key: "card", Title: "", Width: width.Get(), MinWidth: 100, MaxWidth: 4000, NoResize: true, NoMove: true}}
			},
			RowCount:         func() int { a.rRows.Watch(); return max(a.pager.Total(), 0) },
			Text:             func(row, col int) string { return "" },
			RowHeight:        cardPitch,
			HeaderHeight:     1,
			NoRowHeader:      true,
			NoColumnSelect:   true,
			NoLoopSelection:  true,
			NoResizeColumns:  true,
			NoMoveColumns:    true,
			RenderCell:       render,
			RenderHeaderCell: func(cx *core.Scope, _ func() datatable.HeaderCell) core.Node { return core.View{} },
			Selection: func() datatable.Selection {
				a.rSel.Watch()
				if a.selRow < 0 {
					return datatable.NoTableSelection
				}
				return datatable.Selection{Row: a.selRow, Column: -1}
			},
			Empty: core.Text{Content: "No items match.", TestID: "empty"},
			// The look: nothing but padding; the card draws its own frame and
			// selection border.
			StylesFunc: ui.Themed(cx, func() datatable.Styles {
				return datatable.Styles{Cell: core.Style{Padding: core.Edges{Left: 8, Right: 8, Top: 4, Bottom: 4}}}
			}),
			StyleFunc: func() core.Style {
				return core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: pt(0), Background: colors.Background(), Color: colors.Text()}
			},
			OnLayout: func(b core.Bounds) {
				if w := b.Size.Width - 18; w > 100 && w != width.GetUntracked() {
					width.Set(w)
				}
			},
			OnSelectionChange: func(s datatable.Selection) {
				if s.Row >= 0 && s.Row != a.selRow {
					a.requestSelect(s.Row)
				}
				a.rSel.Bump()
			},
			OnVisibleRowsChange: func(start, end int) {
				if a.viewMode == 1 {
					a.setVisible(start, end-1)
				}
			},
		}},
	}
}

// ---- detail form ---------------------------------------------------------

func detailView(cx *core.Scope, a *app) core.Node {
	colors := ui.ColorsOf(cx)
	dis := func() bool { a.rForm.Watch(); return !a.f.has }

	errNode := func(field string) core.Node {
		return core.Show{
			When: func() bool { a.rForm.Watch(); return a.f.errs[field] != "" },
			Then: func(*core.Scope) core.Node {
				return core.Text{ContentFunc: func() string { a.rForm.Watch(); return a.f.errs[field] }, TestID: "err-" + field,
					Style: core.Style{FontSize: 12}, StyleFunc: ui.Colored(colors.Danger)}
			},
		}
	}
	row := func(label string, field string, control core.Node) core.Node {
		kids := []core.Node{control}
		if field != "" {
			kids = append(kids, errNode(field))
		}
		return core.View{Style: core.Style{Direction: core.Row, Gap: 8, AlignItems: core.AlignStart}, Children: []core.Node{
			core.Text{Content: label, Style: core.Style{Width: pt(86), FlexShrink: 0, Padding: core.Edges{Top: 6}}, StyleFunc: ui.Colored(colors.Muted)},
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: pt(0), Gap: 2, AlignItems: core.AlignStretch}, Children: kids},
		}}
	}
	ro := func(f func() string, id string) core.Node {
		return core.Text{ContentFunc: func() string { a.rForm.Watch(); return f() }, TestID: id, Style: core.Style{Padding: core.Edges{Top: 6}}}
	}
	edit := func(fn func(f *formFields)) {
		fn(&a.f.cur)
		a.rForm.Bump()
	}

	statusRadios := []core.Node{}
	for _, s := range gx.AllStatuses {
		s := s
		statusRadios = append(statusRadios, ui.Radio{
			Text: s.String(), TestID: "status-" + s.String(), Disabled: dis,
			Checked:  func() bool { a.rForm.Watch(); return a.f.cur.status == s },
			OnChange: func(bool) { edit(func(f *formFields) { f.status = s }) },
		})
	}
	catOpts := []ui.Option[gx.Category]{}
	for _, c := range gx.AllCategories {
		catOpts = append(catOpts, ui.Option[gx.Category]{Value: c, Label: c.String()})
	}

	fields := []core.Node{
		core.Show{When: func() bool { a.rForm.Watch(); return !a.f.has },
			Then: func(*core.Scope) core.Node {
				return core.Text{Content: "Select an item to edit.", TestID: "no-selection", StyleFunc: ui.Colored(colors.Muted)}
			}},
		row("ID", "", ro(func() string { return idText(a) }, "f-id")),
		row("SKU", "", ro(func() string { return a.f.item.SKU }, "f-sku")),
		row("Name", "name", ui.Input{
			Label: "Name", TestID: "f-name", Disabled: dis,
			Value:    func() string { a.rForm.Watch(); return a.f.cur.name },
			OnChange: func(s string) { edit(func(f *formFields) { f.name = s }) },
		}),
		row("Category", "", ui.Select[gx.Category]{
			Options: catOpts, Label: "Category", TestID: "f-category", Disabled: dis,
			Value:    func() gx.Category { a.rForm.Watch(); return a.f.cur.cat },
			OnChange: func(c gx.Category) { edit(func(f *formFields) { f.cat = c }) },
		}),
		row("Price", "priceCents", ui.Input{
			Label: "Price", TestID: "f-price", Disabled: dis,
			Value:    func() string { a.rForm.Watch(); return a.f.cur.price },
			OnChange: func(s string) { edit(func(f *formFields) { f.price = s }) },
		}),
		row("Quantity", "quantity", ui.NumberInput{
			Label: "Quantity", TestID: "f-qty", Disabled: dis, Min: 0, Max: 1_000_000, Step: 1,
			Value:    func() float64 { a.rForm.Watch(); return a.f.cur.qty },
			OnChange: func(v float64) { edit(func(f *formFields) { f.qty = v }) },
		}),
		row("Status", "", ui.RadioGroup{Horizontal: true, Label: "Status", TestID: "f-status", Children: statusRadios}),
		row("Rating", "rating", core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: []core.Node{
			ui.Slider{
				Label: "Rating", TestID: "f-rating", Disabled: dis, Min: 0, Max: 5, Step: 0.1,
				Style:    core.Style{FlexGrow: 1},
				Value:    func() float64 { a.rForm.Watch(); return a.f.cur.rating },
				OnChange: func(v float64) { edit(func(f *formFields) { f.rating = round1(v) }) },
			},
			core.Text{ContentFunc: func() string { a.rForm.Watch(); return gx.FormatRating(float32(a.f.cur.rating)) }, TestID: "f-rating-text",
				Style: core.Style{Width: pt(28), FlexShrink: 0}},
		}}),
		row("Featured", "", ui.Checkbox{
			Label: "Featured", TestID: "f-featured", Disabled: dis,
			Checked:  func() bool { a.rForm.Watch(); return a.f.cur.featured },
			OnChange: func(v bool) { edit(func(f *formFields) { f.featured = v }) },
		}),
		row("Restock date", "restockDate", core.View{Style: core.Style{Direction: core.Row, Gap: 8, AlignItems: core.AlignCenter}, Children: []core.Node{
			ui.DatePicker{
				Label: "Restock date", TestID: "f-date", Disabled: dis, DateFormat: "%Y-%m-%d", Placeholder: "No date",
				Style: core.Style{FlexGrow: 1},
				Value: func() base.DateValue {
					a.rForm.Watch()
					if a.f.cur.hasDate {
						return base.DateValue{Start: a.f.cur.date}
					}
					return base.DateValue{}
				},
				OnChange: func(v base.DateValue) {
					edit(func(f *formFields) {
						f.hasDate = v.IsSet()
						if f.hasDate {
							y, m, d := v.Start.Date()
							f.date = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
						} else {
							f.date = time.Time{}
						}
					})
				},
			},
			ui.Button{Text: "Clear", TestID: "f-date-clear", Variant: ui.ButtonSecondary, Disabled: dis,
				OnPress: func() { edit(func(f *formFields) { f.hasDate = false; f.date = time.Time{} }) }},
		}}),
		row("Updated", "", ro(func() string {
			if !a.f.has {
				return ""
			}
			return gx.FormatUpdated(a.f.item.UpdatedAt)
		}, "f-updated")),
		row("Notes", "notes", ui.Input{
			Label: "Notes", TestID: "f-notes", Multiline: true, Disabled: dis,
			Style:    core.Style{Height: pt(92)},
			Value:    func() string { a.rForm.Watch(); return a.f.cur.notes },
			OnChange: func(s string) { edit(func(f *formFields) { f.notes = s }) },
		}),
	}

	dirty := func() bool { a.rForm.Watch(); return a.f.dirty() }
	buttons := core.View{Style: core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 8, AlignItems: core.AlignCenter}, Children: []core.Node{
		ui.Button{Text: "Save", TestID: "save", Disabled: func() bool { return !dirty() }, OnPress: a.save},
		ui.Button{Text: "Revert", TestID: "revert", Variant: ui.ButtonSecondary, Disabled: func() bool { return !dirty() }, OnPress: a.revert},
		ui.Button{Text: "Copy Row", TestID: "copy-row", Variant: ui.ButtonOutline, Disabled: dis, OnPress: a.copyRow},
		ui.Button{Text: "Import Notes…", TestID: "import-notes", Variant: ui.ButtonOutline, Disabled: dis, OnPress: a.beginImport},
		core.Show{When: dirty, Then: func(*core.Scope) core.Node {
			return core.Text{Content: "Unsaved changes", TestID: "unsaved", Style: core.Style{FontSize: 12}, StyleFunc: ui.Colored(colors.Muted)}
		}},
	}}

	return core.View{
		TestID: "detail",
		Style:  core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: pt(0), MinWidth: pt(0)},
		StyleFunc: func() core.Style {
			return core.Style{Borders: core.Edges{Left: 1}, BorderColor: colors.Border()}
		},
		Children: []core.Node{core.Scroll{
			Style: core.Style{FlexGrow: 1, FlexShrink: 1},
			Children: []core.Node{core.View{
				Style:    core.Style{Padding: core.All(12), Gap: 10},
				Children: append(fields, buttons),
			}},
		}},
	}
}

func idText(a *app) string {
	if !a.f.has {
		return ""
	}
	return strconv.FormatInt(a.f.item.ID, 10)
}

// ---- status bar -----------------------------------------------------------

func statusBar(cx *core.Scope, a *app) core.Node {
	colors := ui.ColorsOf(cx)
	muted := ui.Colored(colors.Muted)

	exporting := core.Show{
		When: func() bool { a.rExp.Watch(); return a.exp.running },
		Then: func(*core.Scope) core.Node {
			return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: []core.Node{
				ui.Progress{Label: "Export progress", TestID: "export-progress", Style: core.Style{Width: pt(200)},
					Value: func() float32 { a.rExp.Watch(); return float32(a.exportPercent()) }},
				core.Text{ContentFunc: func() string { a.rExp.Watch(); return fmt.Sprintf("%d%%", a.exportPercent()) }, TestID: "export-percent",
					Style: core.Style{Width: pt(36)}},
				ui.Button{Text: "Cancel", TestID: "export-cancel", Variant: ui.ButtonSecondary, Size: ui.SizeSmall, OnPress: a.cancelExport},
			}}
		},
	}

	return core.View{
		TestID: "statusbar",
		Style:  core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12, Padding: core.Symmetric(4, 12), FlexShrink: 0, MinHeight: pt(32)},
		StyleFunc: func() core.Style {
			return core.Style{Borders: core.Edges{Top: 1}, BorderColor: colors.Border()}
		},
		Children: []core.Node{
			core.Text{TestID: "status-text", MaxLines: 1, Style: core.Style{FontSize: 12, FlexShrink: 1, MinWidth: pt(0)}, StyleFunc: muted,
				ContentFunc: func() string { a.rRows.Watch(); a.rVis.Watch(); return a.statusText() }},
			core.Text{TestID: "message", MaxLines: 1, Style: core.Style{FontSize: 12}, ContentFunc: func() string { a.rDlg.Watch(); return a.msg }},
			core.View{Style: core.Style{FlexGrow: 1}},
			exporting,
			core.Text{TestID: "identity", Style: core.Style{FontSize: 12, FlexShrink: 0}, StyleFunc: muted,
				ContentFunc: func() string { return a.identity() }},
			ui.Button{Text: "Host", TestID: "host", Variant: ui.ButtonSecondary, Size: ui.SizeSmall, OnPress: func() { a.hostOpen = true; a.rDlg.Bump() }},
		},
	}
}

// ---- dialogs ---------------------------------------------------------------

func dialogs(cx *core.Scope, a *app) core.Node {
	colors := ui.ColorsOf(cx)
	line := func(s string) core.Node { return core.Text{Content: s} }

	confirm := ui.AlertDialog{
		TestID:       "confirm",
		Open:         func() bool { a.rDlg.Watch(); return a.confirmOpen },
		OnOpenChange: func(open bool, _ base.Reason) {},
		Title:        "Discard unsaved changes?",
		Actions: []ui.AlertAction{
			{Text: "Cancel", Variant: ui.ButtonSecondary, TestID: "confirm-cancel", OnPress: func() { a.confirmAnswer(false) }},
			{Text: "Discard", TestID: "confirm-discard", OnPress: func() { a.confirmAnswer(true) }},
		},
	}
	errorDialog := ui.Dialog{
		TestID: "error",
		Open:   func() bool { a.rDlg.Watch(); return a.errOpen },
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				a.errOpen = false
				a.rDlg.Bump()
			}
		},
		Title:    "Error",
		Children: []core.Node{core.Text{ContentFunc: func() string { a.rDlg.Watch(); return a.errText }, TestID: "error-text"}},
		Footer:   []core.Node{ui.Button{Text: "OK", TestID: "error-ok", OnPress: func() { a.errOpen = false; a.rDlg.Bump() }}},
	}

	about := ui.Dialog{
		TestID: "about",
		Open:   func() bool { a.rDlg.Watch(); return a.aboutOpen },
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				a.aboutOpen = false
				a.rDlg.Bump()
			}
		},
		Title: "About GGUIX",
		Children: []core.Node{core.View{Style: core.Style{Gap: 4}, Children: []core.Node{
			line("GGUIX - NSGo"), line(frameworkVersion()), line(runtime.Version()), line("BuildMarker " + BuildMarker),
		}}},
		Footer: []core.Node{ui.Button{Text: "Close", TestID: "about-close", OnPress: func() { a.aboutOpen = false; a.rDlg.Bump() }}},
	}

	// While any dialog is open, Escape is bound for the whole window. On
	// macOS NSGo only turns Escape into "back" (which closes a modal) when no
	// text field is being edited, and the Search field keeps the keyboard
	// focus when the Host button is clicked; a KeyAlways binding fires
	// wherever the focus is. The binding lives only while a dialog is open,
	// so Escape still reaches popups and text fields the rest of the time.
	escape := core.Show{
		When: func() bool { a.rDlg.Watch(); return a.dialogOpen() },
		Then: func(cx *core.Scope) core.Node {
			cx.OnKeyWith("escape", core.KeyOptions{When: core.KeyAlways}, a.closeDialog)
			// Take the focus out of any text field, so the platform's own
			// Escape-as-back reaches the modal too.
			cx.DismissKeyboard()
			return core.View{}
		},
	}
	return core.View{Style: core.Style{Position: core.Absolute, Width: pt(0), Height: pt(0)}, Children: []core.Node{
		confirm, errorDialog, about, hostDialog(cx, a, colors), escape,
	}}
}

func hostDialog(cx *core.Scope, a *app, colors ui.Colors) core.Node {
	val := func(s string) string {
		if s == "" {
			return "(unavailable)"
		}
		return s
	}
	i := a.info
	rows := [][2]string{
		{"Source", string(i.Source)},
		{"Username", val(i.Username)},
		{"Full name", val(i.FullName)},
		{"UID", val(i.UID)},
		{"Home directory", val(i.HomeDir)},
		{"Hostname", val(i.Hostname)},
		{"OS / Arch", val(strings.TrimSpace(i.OS + " / " + i.Arch))},
		{"PID", val(pidText(i.PID))},
		{"Config directory", val(i.ConfigDir)},
		{"Locale", val(i.Locale)},
	}
	nodes := []core.Node{}
	for _, r := range rows {
		nodes = append(nodes, core.View{Style: core.Style{Direction: core.Row, Gap: 12}, Children: []core.Node{
			core.Text{Content: r[0], Style: core.Style{Width: pt(130), FlexShrink: 0}, StyleFunc: ui.Colored(colors.Muted)},
			core.Text{Content: r[1], TestID: "host-" + strings.ReplaceAll(strings.ToLower(r[0]), " ", "-"),
				Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: pt(0)}, Selectable: core.SelectableOn},
		}})
	}
	for _, n := range i.Notes {
		nodes = append(nodes, core.Text{Content: n, Style: core.Style{FontSize: 12}, StyleFunc: ui.Colored(colors.Muted)})
	}
	if !a.plat.HostActions {
		nodes = append(nodes, core.Text{Content: "Not available in the browser", TestID: "host-actions-note", Style: core.Style{FontSize: 12}, StyleFunc: ui.Colored(colors.Muted)})
	}
	return ui.Dialog{
		TestID: "host-dialog",
		Open:   func() bool { a.rDlg.Watch(); return a.hostOpen },
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				a.hostOpen = false
				a.rDlg.Bump()
			}
		},
		Title:    "Host Info",
		Children: []core.Node{core.View{Style: core.Style{Gap: 6, MinWidth: pt(420)}, Children: nodes}},
		Footer: []core.Node{
			ui.Button{Text: "Reveal Config Folder", TestID: "reveal-config", Variant: ui.ButtonSecondary,
				Disabled: func() bool { return !a.plat.HostActions }, OnPress: a.revealConfig},
			ui.Button{Text: "Reveal Last Export", TestID: "reveal-export", Variant: ui.ButtonSecondary,
				Disabled: func() bool { a.rExp.Watch(); return !a.plat.HostActions || a.exp.lastPath == "" }, OnPress: a.revealExport},
			ui.Button{Text: "Close", TestID: "host-close", OnPress: func() { a.hostOpen = false; a.rDlg.Bump() }},
		},
	}
}

func pidText(p int) string {
	if p == 0 {
		return ""
	}
	return strconv.Itoa(p)
}

func (a *app) revealConfig() {
	if a.plat.OpenFolder == nil || a.info.ConfigDir == "" {
		return
	}
	if err := a.plat.OpenFolder(a.info.ConfigDir); err != nil {
		a.showError(err.Error())
	}
}

func (a *app) revealExport() {
	if a.plat.RevealFile == nil || a.exp.lastPath == "" {
		return
	}
	if err := a.plat.RevealFile(a.exp.lastPath); err != nil {
		a.showError(err.Error())
	}
}

func (a *app) dialogOpen() bool {
	return a.confirmOpen || a.errOpen || a.aboutOpen || a.hostOpen
}

// closeDialog is Escape for the topmost app dialog: the confirm dialog's
// Cancel, else the error, about or host dialog closed.
func (a *app) closeDialog() {
	switch {
	case a.confirmOpen:
		a.confirmAnswer(false)
	case a.errOpen:
		a.errOpen = false
	case a.aboutOpen:
		a.aboutOpen = false
	case a.hostOpen:
		a.hostOpen = false
	default:
		return
	}
	a.rDlg.Bump()
}
