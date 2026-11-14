package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"

	"gguix/core"
)

// Table columns, in order. The id is the sort field the header selects.
var tableCols = []struct {
	title string
	id    core.SortField
	width float32
	align ui.Align
}{
	{"ID", core.SortID, 80, ui.End},
	{"SKU", core.SortSKU, 110, ui.Start},
	{"Name", core.SortName, 260, ui.Start},
	{"Category", core.SortCategory, 110, ui.Start},
	{"Price", core.SortPrice, 100, ui.End},
	{"Qty", core.SortQuantity, 80, ui.End},
	{"Stock", "", 120, ui.Start},
	{"Status", core.SortStatus, 120, ui.Start},
	{"Rating", core.SortRating, 110, ui.Start},
	{"Updated", core.SortUpdated, 150, ui.Start},
	{"Featured", core.SortFeatured, 80, ui.Center},
}

func (a *app) columns() []ui.TableColumn {
	cols := make([]ui.TableColumn, len(tableCols))
	for i, tc := range tableCols {
		title := tc.title
		if tc.id != "" && tc.id == a.sortField {
			if a.sortDesc {
				title += " ▼"
			} else {
				title += " ▲"
			}
		}
		cols[i] = ui.TableColumn{Title: title, ID: tc.title, Width: tc.width, Align: tc.align, Sortable: tc.id != "", MinWidth: 40}
	}
	return cols
}

// view builds the whole window.
func (a *app) view(c *ui.Context) {
	a.frames++
	a.applyTheme(c)
	a.syncForm()
	if !a.searchAt.IsZero() && !c.Now().Before(a.searchAt) {
		a.applySearchNow()
	}
	if !a.msgUntil.IsZero() && c.Now().After(a.msgUntil) && a.msg != "" {
		a.msg = ""
	}
	if a.msg != "" {
		c.After(time.Until(a.msgUntil) + 10*time.Millisecond)
	}
	a.platformShortcuts(c)
	if a.clipOut != "" {
		c.WriteClipboard(a.clipOut)
		a.clipOut = ""
	}
	if a.menuSync != nil {
		a.menuSync()
	}
	ui.Column(c).Fill().Children(func() {
		a.toolbar(c)
		ui.Divider(c)
		ui.SplitPercent(c, &a.splitPct, func() { a.mainList(c) }, func() { a.detail(c) }).Grow(1)
		ui.Divider(c)
		a.statusBar(c)
	})
	a.dialogs(c)
	a.syncPrefs()
}

// applyTheme chooses the light or dark theme. System follows the desktop,
// which is what the context starts with every frame.
func (a *app) applyTheme(c *ui.Context) {
	sys := c.Theme()
	switch a.themeSel {
	case "Light":
		c.SetTheme(ui.LightTheme().WithPreferences(c.Preferences()))
		a.dark = false
	case "Dark":
		c.SetTheme(ui.DarkTheme().WithPreferences(c.Preferences()))
		a.dark = true
	default:
		a.dark = sys.Dark
	}
	a.pal = core.Colors(a.dark)
}

func (a *app) toolbar(c *ui.Context) {
	ui.Row(c).Gap(8).Padding(8, 12).AlignItems(ui.Center).Children(func() {
		// Search, applied 250 ms after the last keystroke.
		search := ui.TextInput(c.Key("search"), &a.search).Placeholder("Search name or SKU").Label("Search").Width(280)
		if a.focusSrch {
			a.focusSrch = false
			search.Focus()
		}
		if search.Changed() {
			if a.bench {
				a.applySearchNow()
			} else {
				a.searchAt = c.Now().Add(250 * time.Millisecond)
				c.After(250 * time.Millisecond)
			}
		}

		// Category filter.
		n := 0
		for _, on := range a.cats {
			if on {
				n++
			}
		}
		label := "Categories"
		if n > 0 {
			label = fmt.Sprintf("Categories (%d)", n)
		}
		btn := ui.Button(c.Key("cats"), label)
		if btn.Clicked() {
			a.catsOpen = !a.catsOpen
		}
		changed := false
		ui.Popover(c, btn, &a.catsOpen, func() {
			ui.Column(c).Gap(6).Padding(8).Children(func() {
				for _, cat := range core.AllCategories {
					if ui.Checkbox(c, &a.cats[cat], cat.String()).Changed() {
						changed = true
					}
				}
				if ui.Button(c, "Clear").Clicked() {
					a.cats = [8]bool{}
					changed = true
				}
			})
		})
		if changed {
			a.rebuildQuery(true)
		}

		// Status filter.
		if ui.Select(c.Key("status"), &a.statusSel, statusChoices).Label("Status").Changed() {
			a.rebuildQuery(true)
		}

		// Go to row.
		goTo := ui.TextInput(c.Key("goto"), &a.goTo).Placeholder("Go to row").Label("Go to row").Width(110)
		submit := goTo.Submitted()
		if ui.Button(c, "Go").Clicked() || submit {
			if n, ok := parseRowNumber(a.goTo); ok {
				a.goToRow(n)
			}
		}

		// View switcher.
		prev := a.viewMode
		ui.Segmented(c.Key("view"), &a.viewMode, "Table", "Cards").Label("View")
		if a.viewMode != prev {
			a.viewChanged()
		}

		ui.Spacer(c)
		ui.Button(c.Key("export"), "Export CSV").Disabled(a.exp.running).OnClick(a.beginExport)
		ui.Select(c.Key("theme"), &a.themeSel, themeChoices).Label("Theme").Width(100)
	})
}

// viewChanged keeps the selection in view after switching views.
func (a *app) viewChanged() {
	if a.selRow >= 0 {
		a.requestScroll(a.selRow)
	} else {
		a.requestScroll(0)
	}
}

// mainList is the table or the cards, with the empty message.
func (a *app) mainList(c *ui.Context) {
	ui.Column(c).Grow(1).Children(func() {
		prev := a.selRow
		var st *ui.ListState
		if a.viewMode == 0 {
			st = &a.table
			a.tableView(c)
		} else {
			st = &a.cards
			a.cardsView(c)
		}
		// A click or key moved the choice: ask before leaving edits.
		if a.selRow != prev {
			row := a.selRow
			a.selRow = prev
			a.requestSelect(row)
		}
		a.afterList(c, st)
	})
}

// afterList scrolls the list as asked and reports its visible rows to the
// pager.
func (a *app) afterList(c *ui.Context, st *ui.ListState) {
	total := a.pager.Total()
	if a.scrollTo >= 0 && total >= 0 {
		st.ScrollTo(min(a.scrollTo, max(total-1, 0)), ui.Start)
		a.scrollTo = -1
	}
	first, last := st.Visible()
	if a.wantFrames > 0 {
		a.wantFrames--
		a.pager.SetVisible(a.visFirst, a.visLast)
		c.AnimationFrame()
		return
	}
	if last >= first && last >= 0 {
		a.visFirst, a.visLast, a.visCount = first, last, last-first+1
		a.pager.SetVisible(first, last)
	}
}

func (a *app) emptyMessage(c *ui.Context) {
	if a.pager.Total() == 0 {
		ui.Text(c, "No items match.").Absolute().Left(0).Right(0).Top(80).TextAlign(ui.Center).TextColor(c.Theme().TextMuted)
	}
}

func (a *app) tableView(c *ui.Context) {
	n := max(a.pager.Total(), 0)
	cols := a.columns()
	a.table.Selected = &a.selRow
	a.table.Sort = &a.sortDummy
	ui.Box(c).Grow(1).Children(func() {
		ui.Table(c.Key("table"), &a.table, cols, n, func(row, col int) { a.cell(c, row, col) }).Grow(1).Label("Items")
		a.emptyMessage(c)
	})
	// A header click: toggle ourselves, and keep the table's own arrow off.
	if col := a.sortDummy.Column; col != "" {
		for _, tc := range tableCols {
			if tc.title == col && tc.id != "" {
				a.setSort(tc.id)
			}
		}
		a.sortDummy = ui.SortOrder{}
	}
}

func (a *app) setSort(f core.SortField) {
	if a.sortField == f {
		a.sortDesc = !a.sortDesc
	} else {
		a.sortField, a.sortDesc = f, false
	}
	a.rebuildQuery(true)
}

func (a *app) cell(c *ui.Context, row, col int) {
	it, ok := a.pager.Row(row)
	if !ok {
		ui.Text(c, "…")
		return
	}
	switch col {
	case 0:
		ui.Text(c, strconv.FormatInt(it.ID, 10)).SingleLine()
	case 1:
		ui.Text(c, it.SKU).SingleLine()
	case 2:
		ui.Text(c, it.Name).SingleLine()
	case 3:
		ui.Text(c, it.Category.String()).SingleLine()
	case 4:
		ui.Text(c, core.FormatPrice(it.PriceCents)).SingleLine()
	case 5:
		ui.Text(c, strconv.Itoa(int(it.Quantity))).SingleLine()
	case 6:
		a.stockBarOnly(c, it.Quantity)
	case 7:
		a.badge(c, it.Status)
	case 8:
		a.stars(c, it.Rating)
	case 9:
		ui.Text(c, core.FormatUpdated(it.UpdatedAt)).SingleLine()
	case 10:
		if it.Featured {
			ui.Text(c, "✓").SingleLine()
		}
	}
}

// stockBarOnly is the table's Stock column: the bar (the quantity has a
// column of its own).
func (a *app) stockBarOnly(c *ui.Context, qty int32) {
	ui.Row(c).AlignItems(ui.Center).Children(func() { a.stockBar(c, qty) })
}

// ---- cards -----------------------------------------------------------

func (a *app) cardsView(c *ui.Context) {
	n := max(a.pager.Total(), 0)
	a.cards.Selected = &a.selRow
	ui.Box(c).Grow(1).Children(func() {
		ui.List(c.Key("cards"), &a.cards, n, func(i int) { a.card(c, i) }).Grow(1).Gap(8).Padding(8).Label("Items")
		a.emptyMessage(c)
	})
}

func (a *app) card(c *ui.Context, i int) {
	t := c.Theme()
	bw, bc := float32(1), t.Border
	if a.selRow == i {
		bw, bc = 2, t.Accent
	}
	it, ok := a.pager.Row(i)
	ui.Box(c).Row().Height(96).Padding(12).Gap(12).Radius(t.Radius).Background(t.Surface).TextColor(t.Text).Border(bw, bc).
		AlignItems(ui.Center).Children(func() {
		if !ok {
			ui.Text(c, "Loading…").Bold()
			return
		}
		a.tile(c, it.Category)
		ui.Column(c).Grow(1).Gap(1).MinWidth(0).Children(func() {
			ui.Text(c, it.Name).Bold().SingleLine()
			ui.Text(c, it.SKU+" · "+it.Category.String()).SingleLine().TextColor(t.TextMuted)
			ui.Text(c, core.NotesPreview(it.Notes)).SingleLine().TextColor(t.TextMuted)
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				a.badge(c, it.Status)
				if it.Featured {
					a.chip(c)
				}
			})
		})
		ui.Column(c).Width(160).Shrink(0).Gap(2).AlignItems(ui.End).Children(func() {
			ui.Text(c, core.FormatPrice(it.PriceCents)).Bold().FontSize(t.FontSize + 5)
			a.stars(c, it.Rating)
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, core.StockLabel(it.Quantity)).SingleLine().TextColor(t.TextMuted)
				a.stockBar(c, it.Quantity)
			})
		})
	})
}

// ---- detail form -----------------------------------------------------

func (a *app) detail(c *ui.Context) {
	f := &a.f
	dis := !f.has
	t := c.Theme()
	ui.Column(c).Fill().Children(func() {
		ui.Scroll(c).Grow(1).Padding(12).Gap(8).Children(func() {
			if dis {
				ui.Text(c, "Select an item to edit.").TextColor(t.TextMuted)
			}
			ui.Form(c, func() {
				ui.Field(c, "ID", func() { ui.Text(c, idText(f)) })
				ui.Field(c, "SKU", func() { ui.Text(c, f.item.SKU) })
				ui.Field(c, "Name", func() {
					ui.TextInput(c.Key("f-name"), &f.cur.name).Disabled(dis)
				}).Error(f.errs["name"])
				ui.Field(c, "Category", func() {
					ui.Select(c.Key("f-cat"), &f.cur.cat, categoryNames()).Disabled(dis)
				})
				ui.Field(c, "Price", func() {
					ui.TextInput(c.Key("f-price"), &f.cur.price).Disabled(dis)
				}).Error(f.errs["priceCents"])
				ui.Field(c, "Quantity", func() {
					ui.NumberInput(c.Key("f-qty"), &f.cur.qty, 0, 1_000_000, 1).Disabled(dis)
				}).Error(f.errs["quantity"])
				ui.Field(c, "Status", func() {
					ui.RadioGroup(c, func() {
						for _, s := range core.AllStatuses {
							ui.Radio(c, &f.cur.status, s, s.String()).Disabled(dis)
						}
					}).Row().Gap(12)
				})
				ui.Field(c, "Rating", func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Slider(c.Key("f-rating"), &f.cur.rating, 0, 5).Grow(1).Disabled(dis)
						f.cur.rating = round1(f.cur.rating)
						ui.Text(c, core.FormatRating(float32(f.cur.rating))).Width(28)
					})
				}).Error(f.errs["rating"])
				ui.Field(c, "Featured", func() {
					ui.Checkbox(c.Key("f-feat"), &f.cur.featured, "").Disabled(dis)
				})
				ui.Field(c, "Restock date", func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if f.cur.hasDate {
							ui.DateInput(c.Key("f-date"), &f.cur.date).Disabled(dis).Label("Restock date")
						} else if ui.Button(c, "Set date").Disabled(dis).Clicked() {
							y, m, d := time.Now().UTC().Date()
							f.cur.hasDate, f.cur.date = true, time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
						}
						if ui.Button(c, "Clear").Disabled(dis || !f.cur.hasDate).Clicked() {
							f.cur.hasDate = false
							f.cur.date = time.Time{}
						}
					})
				}).Error(f.errs["restockDate"])
				ui.Field(c, "Updated", func() { ui.Text(c, updatedText(f)) })
				ui.Field(c, "Notes", func() {
					ui.TextArea(c.Key("f-notes"), &f.cur.notes).Lines(4, 4).Disabled(dis)
				}).Error(f.errs["notes"])
			})
		})
		ui.Divider(c)
		ui.Row(c).Padding(8, 12).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			dirty := f.dirty()
			ui.PrimaryButton(c, "Save").Disabled(!dirty).OnClick(a.save)
			ui.Button(c, "Revert").Disabled(!dirty).OnClick(a.revert)
			ui.Button(c, "Copy Row").Disabled(dis).OnClick(a.copyRow)
			ui.Button(c, "Import Notes…").Disabled(dis).OnClick(a.beginImport)
			if dirty {
				ui.Text(c, "Unsaved changes").TextColor(t.TextMuted)
			}
		})
	})
}

func idText(f *form) string {
	if !f.has {
		return ""
	}
	return strconv.FormatInt(f.item.ID, 10)
}

func updatedText(f *form) string {
	if !f.has {
		return ""
	}
	return core.FormatUpdated(f.item.UpdatedAt)
}

func categoryNames() []string {
	out := make([]string, len(core.AllCategories))
	for i, c := range core.AllCategories {
		out[i] = c.String()
	}
	return out
}

func (a *app) beginImport() {
	if a.plat.OpenTextFile == nil {
		a.showError("Importing a file is not available here.")
		return
	}
	go func() {
		data, ok, err := a.plat.OpenTextFile()
		a.post(func() {
			if err != nil {
				a.showError(err.Error())
			} else if ok {
				a.importNotes(data)
			}
		})
	}()
}

// ---- status bar and dialogs -------------------------------------------

func (a *app) statusBar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Padding(4, 12).Gap(12).AlignItems(ui.Center).Children(func() {
		text := a.statusText()
		if a.msg != "" {
			text = a.msg
		}
		ui.Text(c, text).SingleLine().TextColor(t.TextMuted).Grow(1)
		if a.exp.running {
			ui.Progress(c, float64(a.exp.done)/float64(max(a.exp.total, 1))).Width(200).Label("Export progress")
			ui.Text(c, fmt.Sprintf("%d%%", a.exportPercent())).Width(36)
			ui.Button(c, "Cancel").OnClick(a.cancelExport)
		}
		ui.Text(c, a.identity()).SingleLine()
		if ui.Button(c, "Host").Clicked() {
			a.hostOpen = true
		}
	})
}

func (a *app) dialogs(c *ui.Context) {
	if r := ui.AlertDialog(c, &a.confirmOpen, "Discard unsaved changes?", "", "Cancel", "Discard"); r != -1 {
		if r == 1 && a.confirmFn != nil {
			a.confirmFn()
		} else if a.confirmNo != nil {
			a.confirmNo()
		}
		a.confirmFn, a.confirmNo = nil, nil
	}
	if ui.AlertDialog(c, &a.errOpen, "Error", a.errText, "OK") != -1 {
		a.errText = ""
	}
	ui.Modal(c, &a.aboutOpen, func() {
		ui.Column(c).Gap(6).Padding(8).Children(func() {
			for _, l := range a.aboutLines() {
				ui.Text(c, l)
			}
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.PrimaryButton(c, "Close").Clicked() {
					a.aboutOpen = false
				}
			})
		})
	})
	ui.Modal(c, &a.hostOpen, func() { a.hostPanel(c) })
}

func (a *app) aboutLines() []string {
	return []string{"GGUIX - MyGo", "MyGo " + mygoVersion, runtime.Version(), "BuildMarker " + BuildMarker}
}

const mygoVersion = "v0.9.0"

func (a *app) hostPanel(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Gap(6).Padding(8).Width(520).Children(func() {
		ui.Text(c, "Host").Bold().FontSize(t.FontSize + 3)
		for _, r := range a.info.Rows() {
			ui.Row(c).Gap(12).Children(func() {
				ui.Text(c, r.Label).Width(130).Shrink(0).TextColor(t.TextMuted)
				ui.Text(c, r.Value).Grow(1).Selectable()
			})
		}
		for _, n := range a.info.Notes {
			ui.Text(c, n).TextColor(t.TextMuted)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			ok := a.plat.HostActions && a.info.Available()
			rc := ui.Button(c, "Reveal Config Folder").Disabled(!ok)
			lx := ui.Button(c, "Reveal Last Export").Disabled(!ok || a.exp.lastPath == "")
			if !ok {
				rc.Tooltip("Not available in the browser")
				lx.Tooltip("Not available in the browser")
			}
			if rc.Clicked() {
				a.reveal(a.info.ConfigDir)
			}
			if lx.Clicked() {
				a.reveal(a.exp.lastPath)
			}
			if ui.PrimaryButton(c, "Close").Clicked() {
				a.hostOpen = false
			}
		})
	})
}

func (a *app) reveal(path string) {
	if a.plat.Reveal == nil || path == "" {
		return
	}
	if err := a.plat.Reveal(path); err != nil {
		a.showError(err.Error())
	}
}
