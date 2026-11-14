package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	glyph "github.com/go-gui-org/go-glyph"
	"github.com/go-gui-org/go-gui/gui"

	"gguix/core"
)

// IDs that code names. A leaf holding ":" is absolute in go-gui, so these
// are the effective IDs wherever the widgets sit in the tree.
const (
	tableID  = "gg:table"
	cardsID  = "gg:cards"
	searchID = "gg:search"
	tableBox = "gg:tablebox"
	cardsBox = "gg:cardsbox"
)

// Row heights. go-gui's Table/ListBox rows are text height plus the
// list item padding; 28 px matches that for the default theme. Cards are
// 96 px plus the 8 px gap.
const (
	rowH  float32 = 28
	cardH float32 = 96
	cardG float32 = 8
)

type column struct {
	title string
	field core.SortField
	w     float32
	align gui.HorizontalAlign
}

var columns = []column{
	{"ID", core.SortID, 80, gui.HAlignEnd},
	{"SKU", core.SortSKU, 110, gui.HAlignStart},
	{"Name", core.SortName, 260, gui.HAlignStart},
	{"Category", core.SortCategory, 110, gui.HAlignStart},
	{"Price", core.SortPrice, 100, gui.HAlignEnd},
	{"Qty", core.SortQuantity, 80, gui.HAlignEnd},
	{"Stock", core.SortQuantity, 120, gui.HAlignStart},
	{"Status", core.SortStatus, 120, gui.HAlignStart},
	{"Rating", core.SortRating, 110, gui.HAlignStart},
	{"Updated", core.SortUpdated, 150, gui.HAlignStart},
	{"Featured", core.SortFeatured, 80, gui.HAlignCenter},
}

func tableWidth() float32 {
	var w float32
	for _, c := range columns {
		w += c.w
	}
	return w
}

// headerLabel adds the sort arrow to the active column.
func (a *app) headerLabel(c column) string {
	if c.title == "Stock" || a.sortField != c.field {
		return c.title
	}
	if a.sortDesc {
		return c.title + " ▼"
	}
	return c.title + " ▲"
}

// frameworkVersion reads go-gui's module version from the build info.
func frameworkVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/go-gui-org/go-gui" {
				return d.Version
			}
		}
	}
	return gui.Version
}

// mainView is the window's view function. It runs on the frame thread
// every time the layout is rebuilt.
func mainView(w *gui.Window) gui.View {
	a := gui.State[app](w)
	a.syncForm()
	a.pal = core.Colors(a.isDark())
	if a.msg != "" && time.Now().After(a.msgUntil) {
		a.msg = ""
	}
	a.syncPrefs()
	a.platformFrame(w)
	t := w.Theme()

	var list gui.View
	if a.viewMode == 0 {
		list = a.tableView(w)
	} else {
		list = a.cardsView(w)
	}
	return gui.Column(gui.ContainerCfg{
		Sizing:     gui.FillFill,
		Padding:    gui.NoPadding,
		Spacing:    gui.SpacingSmall,
		SizeBorder: gui.NoBorder,
		Color:      t.ColorBackground,
		Content: []gui.View{
			a.toolbar(w),
			gui.Splitter(gui.SplitterCfg{
				ID:          "gg:split",
				Orientation: gui.SplitterHorizontal,
				Ratio:       gui.SomeF(a.splitPct / 100),
				Sizing:      gui.FillFill,
				OnChange: func(r float32, _ gui.SplitterCollapsed, ctx gui.EventCtx) {
					gui.State[app](ctx.Window).splitPct = r * 100
				},
				First:  gui.SplitterPaneCfg{MinSize: 300, Content: []gui.View{list}},
				Second: gui.SplitterPaneCfg{MinSize: 280, Content: []gui.View{a.detail(w)}},
			}),
			a.statusBar(w),
		},
	})
}

// ---- toolbar ----------------------------------------------------------

func (a *app) toolbar(w *gui.Window) gui.View {
	catLabel := "Categories"
	if n := a.catCount(); n > 0 {
		catLabel = fmt.Sprintf("Categories (%d)", n)
	}
	catContent := []gui.View{
		gui.Button(gui.ButtonCfg{ID: "gg:cats", Label: catLabel, OnClick: func(ctx gui.EventCtx) {
			a.catsOpen = !a.catsOpen
			ctx.Consume()
		}}),
	}
	if a.catsOpen {
		items := []gui.View{}
		for _, c := range core.AllCategories {
			c := c
			items = append(items, gui.Checkbox(gui.ToggleCfg{
				ID: "gg:cat-" + c.String(), Label: c.String(), Selected: a.cats[c],
				OnClick: func(ctx gui.EventCtx) { a.toggleCategory(c); ctx.Consume() },
			}))
		}
		items = append(items, gui.Button(gui.ButtonCfg{ID: "gg:cats-clear", Label: "Clear", OnClick: func(ctx gui.EventCtx) {
			a.clearCategories()
			ctx.Consume()
		}}))
		t := w.Theme()
		catContent = append(catContent, gui.Column(gui.ContainerCfg{
			ID:          "gg:catpop",
			Float:       true,
			FloatAnchor: gui.FloatBottomLeft,
			FloatTieOff: gui.FloatTopLeft,
			FloatZIndex: 10,
			Padding:     t.PaddingMedium,
			Spacing:     gui.SpacingSmall,
			Color:       t.ColorPanel,
			ColorBorder: t.ColorBorder,
			SizeBorder:  gui.BorderThin,
			Radius:      gui.RadiusMedium,
			Content:     items,
		}))
	}
	statusSel := []string{a.statusSel}
	return gui.Row(gui.ContainerCfg{
		Sizing:  gui.FillFit,
		Padding: w.Theme().PaddingSmall,
		Spacing: gui.SpacingSmall,
		VAlign:  gui.VAlignMiddle,
		Content: []gui.View{
			gui.Input(gui.InputCfg{
				ID: searchID, Text: a.search, Placeholder: "Search name or SKU", Width: 280, Sizing: gui.FixedFit,
				OnTextChanged: func(s string, ctx gui.EventCtx) { a.setSearch(s) },
			}),
			gui.Column(gui.ContainerCfg{SizeBorder: gui.NoBorder, Padding: gui.NoPadding, Content: catContent}),
			gui.Select(gui.SelectCfg{
				ID: "gg:status", Items: statusChoices, Selected: statusSel, MinWidth: 150,
				OnSelect: func(v []string, ctx gui.EventCtx) {
					if len(v) > 0 {
						a.setStatusFilter(v[0])
					}
				},
			}),
			gui.Input(gui.InputCfg{
				ID: "gg:goto", Text: a.goTo, Placeholder: "Go to row", Width: 110, Sizing: gui.FixedFit,
				PreTextChange: func(_, p string) (string, bool) {
					return p, strings.Trim(p, "0123456789,") == ""
				},
				OnTextChanged: func(s string, ctx gui.EventCtx) { a.goTo = s },
				OnEnter:       func(ctx gui.EventCtx) { a.goPressed(); ctx.Consume() },
			}),
			gui.Button(gui.ButtonCfg{ID: "gg:go", Label: "Go", OnClick: func(ctx gui.EventCtx) { a.goPressed(); ctx.Consume() }}),
			gui.SegmentedControl(gui.SegmentedControlCfg{
				ID: "gg:viewsw", Items: []string{"Table", "Cards"}, Value: []string{"Table", "Cards"}[a.viewMode],
				OnSelect: func(v string, ctx gui.EventCtx) {
					if v == "Cards" {
						a.setView(1)
					} else {
						a.setView(0)
					}
				},
			}),
			gui.Row(gui.ContainerCfg{Sizing: gui.FillFit, SizeBorder: gui.NoBorder, Padding: gui.NoPadding}),
			gui.Button(gui.ButtonCfg{ID: "gg:export", Label: "Export CSV", Disabled: a.exp.running,
				OnClick: func(ctx gui.EventCtx) { a.beginExport(); ctx.Consume() }}),
			gui.Select(gui.SelectCfg{
				ID: "gg:theme", Items: themeChoices, Selected: []string{a.themeSel}, MinWidth: 110,
				OnSelect: func(v []string, ctx gui.EventCtx) {
					if len(v) > 0 {
						a.setTheme(v[0])
					}
				},
			}),
		},
	})
}

func (a *app) goPressed() {
	if n, ok := parseRowNumber(a.goTo); ok {
		a.goToRow(n)
	}
}

// ---- table and cards --------------------------------------------------

// measure is the AmendLayout hook of the box around a list: it reads the
// list's scroll offset and height and tells the pager what is visible.
func (a *app) measure(listID string, itemH float32) func(gui.EventCtx) {
	return func(ctx gui.EventCtx) {
		if ctx.Layout == nil || ctx.Layout.Shape == nil {
			return
		}
		total := a.pager.Total()
		h := ctx.Layout.Shape.Height
		if listID == tableID {
			h -= rowH // the header
		}
		if total < 0 || h <= 0 {
			return
		}
		off := -ctx.Window.ScrollVerticalOffset(listID)
		first := int(off / itemH)
		last := int((off + h - 1) / itemH)
		if total > 0 {
			first = max(0, min(first, total-1))
			last = max(first, min(last, total-1))
		}
		if a.setVisible(first, last) {
			ctx.Window.QueueCommand(func(w *gui.Window) { w.InvalidateLayout() })
		}
	}
}

func (a *app) listKeys(ctx gui.EventCtx) {
	if ctx.Event == nil {
		return
	}
	switch ctx.Event.KeyCode {
	case gui.KeyUp:
		a.moveSelection(-1)
		ctx.Consume()
	case gui.KeyDown:
		a.moveSelection(1)
		ctx.Consume()
	}
}

func (a *app) emptyMessage() gui.View {
	return gui.Column(gui.ContainerCfg{
		Sizing: gui.FillFill, HAlign: gui.HAlignCenter, VAlign: gui.VAlignMiddle, SizeBorder: gui.NoBorder,
		Content: []gui.View{gui.Text(gui.TextCfg{Text: "No items match."})},
	})
}

func (a *app) tableView(w *gui.Window) gui.View {
	t := w.Theme()
	total := a.pager.Total()
	head := []gui.View{}
	for _, c := range columns {
		c := c
		label := a.headerLabel(c)
		if c.title == "Stock" {
			head = append(head, gui.Row(gui.ContainerCfg{
				Width: c.w, Height: rowH, Sizing: gui.FixedFixed, VAlign: gui.VAlignMiddle, Padding: gui.PadVH(0, 6),
				SizeBorder: gui.NoBorder,
				Content:    []gui.View{gui.Text(gui.TextCfg{Text: label, TextStyle: t.TextStyleLabel})},
			}))
			continue
		}
		head = append(head, gui.Button(gui.ButtonCfg{
			ID: "gg:h-" + c.title, Label: label, Variant: gui.ButtonGhost,
			Width: c.w, Height: rowH, Sizing: gui.FixedFixed, HAlign: gui.Some(c.align),
			OnClick: func(ctx gui.EventCtx) { a.setSort(c.field); ctx.Consume() },
		}))
	}
	var body gui.View
	if total == 0 {
		body = a.emptyMessage()
	} else {
		body = gui.VirtualList(gui.VirtualListCfg{
			ID:            tableID,
			ItemCount:     max(total, 0),
			ItemHeight:    func(int, float32) float32 { return rowH },
			ItemView:      func(i int, _ float32) gui.View { return a.tableRow(w, i) },
			Sizing:        gui.FillFill,
			Padding:       gui.NoPadding,
			OverscanPx:    rowH * 4,
			FocusDisabled: true,
		})
	}
	return gui.Column(gui.ContainerCfg{
		ID: "gg:tablescroll", Sizing: gui.FillFill, ScrollMode: gui.ScrollHorizontalOnly, Scrollable: true,
		Padding: gui.NoPadding, SizeBorder: gui.NoBorder,
		Content: []gui.View{gui.Column(gui.ContainerCfg{
			ID: tableBox, Width: tableWidth(), Sizing: gui.FixedFill, Padding: gui.NoPadding, Spacing: gui.SpacingPx(0),
			SizeBorder: gui.NoBorder, Focusable: true, OnKeyDown: a.listKeys,
			AmendLayout: a.measure(tableID, rowH),
			Content: []gui.View{
				gui.Row(gui.ContainerCfg{Sizing: gui.FillFixed, Height: rowH, Padding: gui.NoPadding, Spacing: gui.SpacingPx(0),
					SizeBorder: gui.NoBorder, Color: t.ColorPanel, Content: head}),
				body,
			},
		})},
	})
}

func (a *app) tableRow(w *gui.Window, i int) gui.View {
	t := w.Theme()
	it, ok := a.pager.Row(i)
	cells := make([]gui.View, 0, len(columns))
	for ci, c := range columns {
		var content gui.View
		if !ok {
			content = gui.Text(gui.TextCfg{Text: "…"})
		} else {
			content = a.cell(w, ci, it, i)
		}
		cells = append(cells, gui.Row(gui.ContainerCfg{
			Width: c.w, Height: rowH, Sizing: gui.FixedFixed, HAlign: c.align, VAlign: gui.VAlignMiddle,
			Padding: gui.PadVH(0, 6), SizeBorder: gui.NoBorder, Clip: true, Content: []gui.View{content},
		}))
	}
	var bg gui.Color
	if i == a.selRow {
		bg = t.ColorSelect
	}
	return gui.Row(gui.ContainerCfg{
		ID: "r" + strconv.Itoa(i), Height: rowH, Sizing: gui.FillFixed, Padding: gui.NoPadding, Spacing: gui.SpacingPx(0),
		SizeBorder: gui.NoBorder, Color: bg, Hover: gui.HoverStyle{Color: t.ColorHover},
		OnClick: func(ctx gui.EventCtx) { a.requestSelect(i); ctx.Consume() },
		Content: cells,
	})
}

func (a *app) cell(w *gui.Window, col int, it core.Item, row int) gui.View {
	txt := func(s string) gui.View { return gui.Text(gui.TextCfg{Text: s}) }
	switch col {
	case 0:
		return txt(strconv.FormatInt(it.ID, 10))
	case 1:
		return txt(it.SKU)
	case 2:
		return txt(it.Name)
	case 3:
		return txt(it.Category.String())
	case 4:
		return txt(core.FormatPrice(it.PriceCents))
	case 5:
		return txt(strconv.Itoa(int(it.Quantity)))
	case 6:
		return gui.Row(gui.ContainerCfg{SizeBorder: gui.NoBorder, Padding: gui.NoPadding, Spacing: gui.SpacingSmall, VAlign: gui.VAlignMiddle,
			Content: []gui.View{a.stockBar(it.Quantity, row), txt(strconv.Itoa(int(it.Quantity)))}})
	case 7:
		return a.badge(w, it.Status)
	case 8:
		return a.starsWithText(it.Rating, row, "t")
	case 9:
		return txt(core.FormatUpdated(it.UpdatedAt))
	case 10:
		if it.Featured {
			return txt("✓")
		}
		return txt("")
	}
	return txt("")
}

func (a *app) cardsView(w *gui.Window) gui.View {
	total := a.pager.Total()
	var body gui.View
	if total == 0 {
		body = a.emptyMessage()
	} else {
		body = gui.VirtualList(gui.VirtualListCfg{
			ID:            cardsID,
			ItemCount:     max(total, 0),
			ItemHeight:    func(int, float32) float32 { return cardH + cardG },
			ItemView:      func(i int, _ float32) gui.View { return a.card(w, i) },
			Sizing:        gui.FillFill,
			Padding:       gui.PadAll(8),
			OverscanPx:    cardH * 2,
			FocusDisabled: true,
		})
	}
	return gui.Column(gui.ContainerCfg{
		ID: cardsBox, Sizing: gui.FillFill, Padding: gui.NoPadding, SizeBorder: gui.NoBorder,
		Focusable: true, OnKeyDown: a.listKeys, AmendLayout: a.measure(cardsID, cardH+cardG),
		Content: []gui.View{body},
	})
}

func (a *app) card(w *gui.Window, i int) gui.View {
	t := w.Theme()
	it, ok := a.pager.Row(i)
	border, bw := t.ColorBorder, gui.BorderThin
	if i == a.selRow {
		border, bw = t.ColorAccent, gui.BorderPx(2)
	}
	frame := func(content []gui.View) gui.View {
		return gui.Column(gui.ContainerCfg{
			Sizing: gui.FillFixed, Height: cardH + cardG, Padding: gui.NewPadding(0, 0, cardG, 0), SizeBorder: gui.NoBorder,
			Content: []gui.View{gui.Row(gui.ContainerCfg{
				ID: "c" + strconv.Itoa(i), Sizing: gui.FillFixed, Height: cardH, Padding: gui.PadAll(12), Spacing: gui.SpacingMedium,
				Radius: gui.RadiusMedium, SizeBorder: bw, ColorBorder: border, Color: t.ColorPanel, VAlign: gui.VAlignMiddle,
				OnClick: func(ctx gui.EventCtx) { a.requestSelect(i); ctx.Consume() },
				Content: content,
			})},
		})
	}
	if !ok {
		return frame([]gui.View{gui.Text(gui.TextCfg{Text: "Loading…"})})
	}
	bold := t.TextStyleDef
	bold.Typeface = glyph.TypefaceBold
	tileText := bold
	tileText.Color = gui.RGB(255, 255, 255)
	price := t.TextStyleTitle
	price.Typeface = glyph.TypefaceBold
	chips := []gui.View{a.badge(w, it.Status)}
	if it.Featured {
		chips = append(chips, a.pill(w, "Featured", a.pal.Featured.Background, a.pal.Featured.Text))
	}
	return frame([]gui.View{
		gui.Column(gui.ContainerCfg{
			Width: 48, Height: 48, Sizing: gui.FixedFixed, Radius: gui.RadiusMedium, SizeBorder: gui.NoBorder,
			Color: hexColor(a.pal.Tile(it.Category)), HAlign: gui.HAlignCenter, VAlign: gui.VAlignMiddle,
			Content: []gui.View{gui.Text(gui.TextCfg{Text: core.CategoryInitial(it.Category), TextStyle: tileText})},
		}),
		gui.Column(gui.ContainerCfg{
			Sizing: gui.FillFill, Padding: gui.NoPadding, Spacing: gui.SpacingPx(2), SizeBorder: gui.NoBorder, Clip: true,
			Content: []gui.View{
				gui.Text(gui.TextCfg{Text: it.Name, TextStyle: bold}),
				gui.Text(gui.TextCfg{Text: it.SKU + " · " + it.Category.String(), TextStyle: t.TextStyleSecondary}),
				gui.Text(gui.TextCfg{Text: core.NotesPreview(it.Notes), TextStyle: t.TextStyleSecondary, Clip: true, Sizing: gui.FillFit}),
				gui.Row(gui.ContainerCfg{Padding: gui.NoPadding, Spacing: gui.SpacingSmall, SizeBorder: gui.NoBorder, Content: chips}),
			},
		}),
		gui.Column(gui.ContainerCfg{
			Width: 160, Sizing: gui.FixedFill, Padding: gui.NoPadding, Spacing: gui.SpacingPx(4), HAlign: gui.HAlignEnd,
			SizeBorder: gui.NoBorder,
			Content: []gui.View{
				gui.Text(gui.TextCfg{Text: core.FormatPrice(it.PriceCents), TextStyle: price}),
				a.starsWithText(it.Rating, i, "c"),
				gui.Row(gui.ContainerCfg{Padding: gui.NoPadding, Spacing: gui.SpacingSmall, SizeBorder: gui.NoBorder, VAlign: gui.VAlignMiddle,
					Content: []gui.View{gui.Text(gui.TextCfg{Text: core.StockLabel(it.Quantity), TextStyle: t.TextStyleSecondary}), a.stockBar(it.Quantity, -1)}}),
			},
		}),
	})
}

// ---- detail form ------------------------------------------------------

func categoryNames() []string {
	out := make([]string, len(core.AllCategories))
	for i, c := range core.AllCategories {
		out[i] = c.String()
	}
	return out
}

func statusNames() []string {
	out := make([]string, len(core.AllStatuses))
	for i, s := range core.AllStatuses {
		out[i] = s.String()
	}
	return out
}

func (a *app) detail(w *gui.Window) gui.View {
	t := w.Theme()
	f := &a.f
	off := !f.has
	cur := &f.cur
	errText := func(field string) gui.View {
		if msg, ok := f.errs[field]; ok {
			return gui.Text(gui.TextCfg{ID: "gg:err-" + field, Text: msg, TextStyle: t.TextStyleError, Mode: gui.TextModeWrap})
		}
		return nil
	}
	field := func(label string, control gui.View, errField string) gui.View {
		right := []gui.View{control}
		if e := errText(errField); e != nil {
			right = append(right, e)
		}
		return gui.Row(gui.ContainerCfg{
			Sizing: gui.FillFit, Padding: gui.NoPadding, Spacing: gui.SpacingMedium, SizeBorder: gui.NoBorder, VAlign: gui.VAlignTop,
			Content: []gui.View{
				gui.Row(gui.ContainerCfg{Width: 100, Sizing: gui.FixedFit, Padding: gui.PadVH(6, 0), SizeBorder: gui.NoBorder,
					Content: []gui.View{gui.Text(gui.TextCfg{Text: label, TextStyle: t.TextStyleLabel})}}),
				gui.Column(gui.ContainerCfg{Sizing: gui.FillFit, Padding: gui.NoPadding, Spacing: gui.SpacingTight, SizeBorder: gui.NoBorder, Content: right}),
			},
		})
	}
	ro := func(id, s string) gui.View {
		return gui.Input(gui.InputCfg{ID: id, Text: s, ReadOnly: true, Disabled: off, Sizing: gui.FillFit, FocusDisabled: true})
	}
	var date time.Time
	if cur.hasDate {
		date = cur.date
	}
	content := []gui.View{}
	if off {
		content = append(content, gui.Text(gui.TextCfg{Text: "Select an item to edit."}))
	}
	content = append(content,
		field("ID", ro("gg:f-id", idText(f)), ""),
		field("SKU", ro("gg:f-sku", skuText(f)), ""),
		field("Name", gui.Input(gui.InputCfg{ID: "gg:f-name", Text: cur.name, Disabled: off, Sizing: gui.FillFit,
			OnTextChanged: func(s string, _ gui.EventCtx) { a.f.cur.name = s }}), "name"),
		field("Category", gui.Select(gui.SelectCfg{ID: "gg:f-cat", Items: categoryNames(), Selected: []string{cur.cat}, Disabled: off, Sizing: gui.FillFit,
			OnSelect: func(v []string, _ gui.EventCtx) {
				if len(v) > 0 {
					a.f.cur.cat = v[0]
				}
			}}), "category"),
		field("Price", gui.Input(gui.InputCfg{ID: "gg:f-price", Text: cur.price, Disabled: off, Sizing: gui.FillFit,
			OnTextChanged: func(s string, _ gui.EventCtx) { a.f.cur.price = s }}), "priceCents"),
		field("Quantity", gui.NumericInput(gui.NumericInputCfg{ID: "gg:f-qty", Text: cur.qty, Disabled: off, Sizing: gui.FillFit,
			Min: gui.Some(0.0), Max: gui.Some(1_000_000.0), Decimals: 0,
			StepCfg:       gui.NumericStepCfg{Step: 1},
			OnTextChanged: func(s string, _ gui.EventCtx) { a.f.cur.qty = s },
			OnValueCommit: func(v gui.Opt[float64], s string, _ gui.EventCtx) {
				if v.IsSet() {
					a.f.cur.qty = strconv.Itoa(int(v.Get(0)))
				} else {
					a.f.cur.qty = s
				}
			}}), "quantity"),
		field("Status", gui.RadioButtonGroupRow(gui.RadioButtonGroupCfg{ID: "gg:f-status", Items: statusNames(), Value: cur.status.String(), Disabled: off,
			SizeBorder: gui.NoBorder, Padding: gui.NoPadding,
			OnSelect: func(v string, _ gui.EventCtx) {
				if s, err := core.ParseStatus(v); err == nil {
					a.f.cur.status = s
				}
			}}), "status"),
		field("Rating", gui.Row(gui.ContainerCfg{Sizing: gui.FillFit, Padding: gui.NoPadding, Spacing: gui.SpacingMedium, SizeBorder: gui.NoBorder, VAlign: gui.VAlignMiddle,
			Content: []gui.View{
				gui.Slider(gui.SliderCfg{ID: "gg:f-rating", Value: float32(cur.rating), Min: 0, Max: 5, Step: 0.1, Disabled: off, Width: 220, Sizing: gui.FixedFit,
					OnChange: func(v float32, _ gui.EventCtx) { a.f.cur.rating = round1(float64(v)) }}),
				gui.Text(gui.TextCfg{Text: core.FormatRating(float32(cur.rating))}),
			}}), "rating"),
		field("Featured", gui.Checkbox(gui.ToggleCfg{ID: "gg:f-featured", Selected: cur.featured, Disabled: off,
			OnClick: func(ctx gui.EventCtx) { a.f.cur.featured = !a.f.cur.featured; ctx.Consume() }}), "featured"),
		field("Restock date", gui.Row(gui.ContainerCfg{Sizing: gui.FillFit, Padding: gui.NoPadding, Spacing: gui.SpacingSmall, SizeBorder: gui.NoBorder, VAlign: gui.VAlignMiddle,
			Content: []gui.View{
				gui.InputDate(gui.InputDateCfg{ID: "gg:f-date", Date: date, Disabled: off, Sizing: gui.FillFit, Placeholder: "YYYY-MM-DD", DateFormat: "YYYY-MM-DD",
					OnSelect: func(v []time.Time, _ gui.EventCtx) {
						if len(v) > 0 {
							y, m, d := v[0].Date()
							a.f.cur.hasDate, a.f.cur.date = true, time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
						}
					}}),
				gui.Button(gui.ButtonCfg{ID: "gg:f-dateclear", Label: "Clear", Disabled: off,
					OnClick: func(ctx gui.EventCtx) { a.f.cur.hasDate, a.f.cur.date = false, time.Time{}; ctx.Consume() }}),
			}}), "restockDate"),
		field("Updated", ro("gg:f-updated", updatedText(f)), ""),
		field("Notes", gui.Input(gui.InputCfg{ID: "gg:f-notes", Text: cur.notes, Mode: gui.InputMultiline, Disabled: off,
			Height: 4*float32(t.TextStyleDef.Size)*1.4 + 12, Sizing: gui.FillFixed,
			OnTextChanged: func(s string, _ gui.EventCtx) { a.setNotes(s) }}), "notes"),
	)
	dirty := f.dirty()
	buttons := []gui.View{
		gui.Button(gui.ButtonCfg{ID: "gg:save", Label: "Save", Variant: gui.ButtonPrimary, Disabled: !dirty,
			OnClick: func(ctx gui.EventCtx) { a.save(); ctx.Consume() }}),
		gui.Button(gui.ButtonCfg{ID: "gg:revert", Label: "Revert", Disabled: !dirty,
			OnClick: func(ctx gui.EventCtx) { a.revert(); ctx.Consume() }}),
		gui.Button(gui.ButtonCfg{ID: "gg:copyrow", Label: "Copy Row", Disabled: off,
			OnClick: func(ctx gui.EventCtx) { a.copyRow(); ctx.Consume() }}),
		gui.Button(gui.ButtonCfg{ID: "gg:import", Label: "Import Notes…", Disabled: off,
			OnClick: func(ctx gui.EventCtx) { a.beginImport(); ctx.Consume() }}),
	}
	if dirty {
		buttons = append(buttons, gui.Text(gui.TextCfg{Text: "Unsaved changes", TextStyle: t.TextStyleSecondary}))
	}
	content = append(content, gui.Row(gui.ContainerCfg{Sizing: gui.FillFit, Padding: gui.PadVH(8, 0), Spacing: gui.SpacingSmall,
		SizeBorder: gui.NoBorder, VAlign: gui.VAlignMiddle, Wrap: true, Content: buttons}))
	return gui.Column(gui.ContainerCfg{
		ID: "gg:detail", Sizing: gui.FillFill, Padding: t.PaddingMedium, Spacing: gui.SpacingSmall, SizeBorder: gui.NoBorder,
		Scrollable: true, ScrollMode: gui.ScrollVerticalOnly,
		Content: content,
	})
}

func idText(f *form) string {
	if !f.has {
		return ""
	}
	return strconv.FormatInt(f.item.ID, 10)
}

func skuText(f *form) string {
	if !f.has {
		return ""
	}
	return f.item.SKU
}

func updatedText(f *form) string {
	if !f.has {
		return ""
	}
	return core.FormatUpdated(f.item.UpdatedAt)
}

func (a *app) beginImport() {
	if !a.f.has {
		return
	}
	if a.plat.OpenTextFile == nil {
		a.showError("Importing a file is not available here.")
		return
	}
	a.plat.OpenTextFile(func(data []byte, ok bool, err error) {
		a.post(func() {
			if err != nil {
				a.showError(err.Error())
			} else if ok {
				a.importNotes(data)
			}
		})
	})
}

// ---- status bar -------------------------------------------------------

func (a *app) statusBar(w *gui.Window) gui.View {
	t := w.Theme()
	text := a.statusText()
	if a.msg != "" {
		text = a.msg
	}
	items := []gui.View{
		gui.Text(gui.TextCfg{ID: "gg:status-text", Text: text, TextStyle: t.TextStyleSecondary}),
		gui.Row(gui.ContainerCfg{Sizing: gui.FillFit, SizeBorder: gui.NoBorder, Padding: gui.NoPadding}),
	}
	if a.exp.running {
		pct := a.exportPercent()
		items = append(items,
			gui.ProgressBar(gui.ProgressBarCfg{ID: "gg:progress", Percent: float32(pct) / 100, Width: 200, Sizing: gui.FixedFit}),
			gui.Text(gui.TextCfg{Text: fmt.Sprintf("%d%%", pct)}),
			gui.Button(gui.ButtonCfg{ID: "gg:cancel", Label: "Cancel", OnClick: func(ctx gui.EventCtx) { a.cancelExport(); ctx.Consume() }}),
		)
	}
	items = append(items,
		gui.Text(gui.TextCfg{ID: "gg:identity", Text: a.identity()}),
		gui.Button(gui.ButtonCfg{ID: "gg:host", Label: "Host", OnClick: func(ctx gui.EventCtx) { a.openHost(); ctx.Consume() }}),
	)
	return gui.Row(gui.ContainerCfg{
		Sizing: gui.FillFit, Padding: t.PaddingSmall, Spacing: gui.SpacingMedium, VAlign: gui.VAlignMiddle,
		SizeBorder: gui.NoBorder, Color: t.ColorPanel, Content: items,
	})
}

// ---- dialogs ----------------------------------------------------------

func (a *app) openAbout() { a.aboutOpen = true; a.dialogsChanged() }
func (a *app) openHost()  { a.hostOpen = true; a.dialogsChanged() }

func (a *app) aboutLines() []string {
	return []string{"GGUIX - go-gui", "go-gui " + frameworkVersion(), "Go " + runtime.Version(), "BuildMarker " + BuildMarker}
}

// hostRows is the host panel's label/value list.
func (a *app) hostRows() [][2]string {
	i := a.info
	v := func(s string) string {
		if s == "" {
			return "(unavailable)"
		}
		return s
	}
	rows := [][2]string{
		{"Source", v(string(i.Source))},
		{"Username", v(i.Username)},
		{"Full name", v(i.FullName)},
		{"UID", v(i.UID)},
		{"Home directory", v(i.HomeDir)},
		{"Hostname", v(i.Hostname)},
		{"OS / Arch", v(i.OS + " / " + i.Arch)},
		{"PID", v(func() string {
			if i.PID == 0 {
				return ""
			}
			return strconv.Itoa(i.PID)
		}())},
		{"Config directory", v(i.ConfigDir)},
		{"Locale", v(i.Locale)},
	}
	for _, n := range i.Notes {
		rows = append(rows, [2]string{"Note", n})
	}
	return rows
}

// showDialogs opens the in-window modal for the highest-priority pending
// dialog. go-gui has one dialog slot; the others wait until it closes.
func (a *app) showDialogs(w *gui.Window) {
	if w.DialogIsVisible() {
		return
	}
	t := w.Theme()
	text := func(s string) gui.View { return gui.Text(gui.TextCfg{Text: s, Mode: gui.TextModeWrap}) }
	btn := func(id, label string, primary bool, fn func()) gui.View {
		v := gui.ButtonSecondary
		if primary {
			v = gui.ButtonPrimary
		}
		return gui.Button(gui.ButtonCfg{ID: id, Label: label, Variant: v, OnClick: func(ctx gui.EventCtx) {
			ctx.Window.DialogDismiss()
			fn()
			ctx.Consume()
			ctx.Window.QueueCommand(a.showDialogs)
		}})
	}
	buttons := func(b ...gui.View) gui.View {
		return gui.Row(gui.ContainerCfg{Sizing: gui.FillFit, HAlign: gui.HAlignEnd, Spacing: gui.SpacingSmall, Padding: gui.NoPadding, SizeBorder: gui.NoBorder, Content: b})
	}
	custom := func(title string, view func(*gui.Window) gui.View) {
		w.Dialog(gui.DialogCfg{DialogType: gui.DialogCustom, Title: title, CustomView: view, EscapeDisabled: true})
	}
	switch {
	case a.confirmOpen:
		custom("Discard unsaved changes?", func(*gui.Window) gui.View {
			return gui.Column(gui.ContainerCfg{SizeBorder: gui.NoBorder, Padding: gui.NoPadding, Spacing: gui.SpacingMedium, Content: []gui.View{
				text("Discard unsaved changes?"),
				buttons(
					btn("gg:dlg-cancel", "Cancel", false, func() {
						a.confirmOpen = false
						if a.confirmNo != nil {
							a.confirmNo()
						}
					}),
					btn("gg:dlg-discard", "Discard", true, func() {
						a.confirmOpen = false
						if a.confirmFn != nil {
							a.confirmFn()
						}
					}),
				),
			}})
		})
	case a.errOpen:
		custom("Error", func(*gui.Window) gui.View {
			return gui.Column(gui.ContainerCfg{SizeBorder: gui.NoBorder, Padding: gui.NoPadding, Spacing: gui.SpacingMedium, Content: []gui.View{
				gui.Text(gui.TextCfg{Text: a.errText, TextStyle: t.TextStyleError, Mode: gui.TextModeWrap}),
				buttons(btn("gg:dlg-ok", "OK", true, func() { a.errOpen = false })),
			}})
		})
	case a.aboutOpen:
		custom("About GGUIX", func(*gui.Window) gui.View {
			c := []gui.View{}
			for _, l := range a.aboutLines() {
				c = append(c, text(l))
			}
			c = append(c, buttons(btn("gg:dlg-ok", "OK", true, func() { a.aboutOpen = false })))
			return gui.Column(gui.ContainerCfg{SizeBorder: gui.NoBorder, Padding: gui.NoPadding, Spacing: gui.SpacingSmall, Content: c})
		})
	case a.hostOpen:
		custom("Host Info", func(*gui.Window) gui.View { return a.hostPanel(w) })
	}
}

func (a *app) hostPanel(w *gui.Window) gui.View {
	t := w.Theme()
	rows := []gui.View{}
	for _, r := range a.hostRows() {
		rows = append(rows, gui.Row(gui.ContainerCfg{Padding: gui.NoPadding, Spacing: gui.SpacingMedium, SizeBorder: gui.NoBorder, Content: []gui.View{
			gui.Row(gui.ContainerCfg{Width: 140, Sizing: gui.FixedFit, Padding: gui.NoPadding, SizeBorder: gui.NoBorder,
				Content: []gui.View{gui.Text(gui.TextCfg{Text: r[0], TextStyle: t.TextStyleLabel})}}),
			gui.Text(gui.TextCfg{Text: r[1]}),
		}}))
	}
	actions := []gui.View{}
	if a.plat.HostActions {
		actions = append(actions,
			gui.Button(gui.ButtonCfg{ID: "gg:reveal-config", Label: "Reveal Config Folder",
				OnClick: func(ctx gui.EventCtx) { a.reveal(a.info.ConfigDir); ctx.Consume() }}),
			gui.Button(gui.ButtonCfg{ID: "gg:reveal-export", Label: "Reveal Last Export", Disabled: a.exp.lastPath == "",
				OnClick: func(ctx gui.EventCtx) { a.reveal(a.exp.lastPath); ctx.Consume() }}),
		)
	} else {
		note := "Not available in the browser"
		if a.info.Source == "server-process" {
			note = "Not available: these would act on the server"
		}
		actions = append(actions,
			gui.Button(gui.ButtonCfg{ID: "gg:reveal-config", Label: "Reveal Config Folder", Disabled: true}),
			gui.Button(gui.ButtonCfg{ID: "gg:reveal-export", Label: "Reveal Last Export", Disabled: true}),
			gui.Text(gui.TextCfg{Text: note, TextStyle: t.TextStyleSecondary}),
		)
	}
	actions = append(actions, gui.Row(gui.ContainerCfg{Sizing: gui.FillFit, SizeBorder: gui.NoBorder, Padding: gui.NoPadding}),
		gui.Button(gui.ButtonCfg{ID: "gg:dlg-close", Label: "Close", Variant: gui.ButtonPrimary, OnClick: func(ctx gui.EventCtx) {
			a.hostOpen = false
			ctx.Window.DialogDismiss()
			ctx.Consume()
			ctx.Window.QueueCommand(a.showDialogs)
		}}))
	rows = append(rows, gui.Row(gui.ContainerCfg{Sizing: gui.FillFit, Padding: gui.PadVH(8, 0), Spacing: gui.SpacingSmall, SizeBorder: gui.NoBorder,
		VAlign: gui.VAlignMiddle, Content: actions}))
	return gui.Column(gui.ContainerCfg{SizeBorder: gui.NoBorder, Padding: gui.NoPadding, Spacing: gui.SpacingSmall, MinWidth: 520, Content: rows})
}

func (a *app) reveal(path string) {
	if path == "" || a.plat.Reveal == nil {
		return
	}
	if err := a.plat.Reveal(path); err != nil {
		a.showError(err.Error())
	}
}

// ---- commands (shortcuts) ---------------------------------------------

// registerCommands installs the shortcuts. They are global so they also
// fire while a text field has focus. The native menu items reference
// them by ID, so menu and keyboard run the same code.
func (a *app) registerCommands(w *gui.Window) {
	m := gui.ShortcutModifier()
	cmd := func(id, label string, key gui.KeyCode, mods gui.Modifier, fn func()) gui.Command {
		return gui.Command{ID: id, Label: label, Global: true, Shortcut: gui.Shortcut{Key: key, Modifiers: mods},
			Execute: func(_ *gui.Event, w *gui.Window) { fn(); w.InvalidateLayout() }}
	}
	_ = w.RegisterCommands(
		cmd("export", "Export CSV…", gui.KeyE, m, a.beginExport),
		cmd("import", "Import Notes…", gui.KeyO, m, a.beginImport),
		cmd("copyrow", "Copy Row", gui.KeyC, m|gui.ModShift, a.copyRow),
		cmd("view.table", "Table", gui.Key1, m, func() { a.setView(0) }),
		cmd("view.cards", "Cards", gui.Key2, m, func() { a.setView(1) }),
		cmd("reload", "Reload", gui.KeyR, m, func() { a.rebuildQuery(true) }),
		cmd("focus.search", "Focus Search", gui.KeyF, m, func() {
			if a.focusFn != nil {
				a.focusFn(searchID)
			}
		}),
		cmd("host", "Host Info…", gui.KeyI, m, a.openHost),
	)
}

// applyTheme makes the window's theme match the choice. System follows
// the OS appearance live through FollowSystemAppearance.
func (a *app) applyTheme(w *gui.Window) {
	switch a.themeSel {
	case "Light":
		w.SetTheme(gui.ThemeLight)
	case "Dark":
		w.SetTheme(gui.ThemeDark)
	default:
		w.FollowSystemAppearance(gui.ThemeLight, gui.ThemeDark)
	}
	w.InvalidateLayout()
}

// bindWindow connects the model's hooks to a window.
func (a *app) bindWindow(w *gui.Window) {
	a.win = w
	a.post = func(f func()) {
		w.QueueCommand(func(w *gui.Window) { f(); w.InvalidateLayout() })
	}
	a.wake = w.InvalidateLayout
	a.scrollFn = func(view, row int) {
		id := tableID
		if view == 1 {
			id = cardsID
		}
		w.ScrollToIndex(id, row)
	}
	a.dialogFn = func() { w.QueueCommand(a.showDialogs) }
	a.clipFn = w.SetClipboard
	a.focusFn = func(id string) { w.QueueCommand(func(w *gui.Window) { w.SetFocus(id) }) }
	a.themeFn = func() { a.applyTheme(w) }
	if ap, ok := w.SystemAppearance(); ok {
		a.sysDark = ap == gui.AppearanceDark
	}
	w.OnSystemAppearance(func(ap gui.Appearance) {
		a.sysDark = ap == gui.AppearanceDark
		w.InvalidateLayout()
	})
	a.registerCommands(w)
	a.applyTheme(w)
}

func hexColor(s string) gui.Color {
	c := core.HexColor(s)
	return gui.RGB(c.R, c.G, c.B)
}
