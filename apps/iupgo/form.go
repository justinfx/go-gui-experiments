package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/iup-go/iup"

	"gguix/core"
)

// formView holds the detail form controls.
type formView struct {
	box, hint                        iup.Ihandle
	id, sku, name, price, qty, upd   iup.Ihandle
	cat, rating, ratingLbl, featured iup.Ihandle
	status                           [3]iup.Ihandle
	date, dateLbl, dateClear, notes  iup.Ihandle
	save, revert, copyRow, importBtn iup.Ihandle
	dirtyLbl                         iup.Ihandle
	errs                             map[string]iup.Ihandle // field -> error row
	errLbl                           map[string]iup.Ihandle
	loading                          bool
	fields                           []iup.Ihandle
}

func (a *app) buildForm() iup.Ihandle {
	f := &a.ui.frm
	f.errs = map[string]iup.Ihandle{}
	f.errLbl = map[string]iup.Ihandle{}

	label := func(s string) iup.Ihandle {
		l := iup.Label(s)
		set(l, "RASTERSIZE", "100x", "ALIGNMENT", "ARIGHT:ACENTER")
		return l
	}
	row := func(l string, c iup.Ihandle) iup.Ihandle {
		h := iup.Hbox(label(l), c)
		set(h, "GAP", "8", "ALIGNMENT", "ACENTER")
		return h
	}
	errRow := func(field string) iup.Ihandle {
		sp := iup.Label("")
		set(sp, "RASTERSIZE", "100x")
		el := iup.Label("")
		set(el, "EXPAND", "HORIZONTAL", "WORDWRAP", "YES", "ELLIPSIS", "NO")
		f.errLbl[field] = el
		h := iup.Hbox(sp, el)
		set(h, "GAP", "8", "FLOATING", "IGNORE", "VISIBLE", "NO")
		f.errs[field] = h
		return h
	}
	text := func(readonly bool) iup.Ihandle {
		t := iup.Text()
		set(t, "EXPAND", "HORIZONTAL")
		if readonly {
			set(t, "READONLY", "YES")
		}
		return t
	}

	f.hint = iup.Label("Select an item to edit.")
	set(f.hint, "EXPAND", "HORIZONTAL", "FLOATING", "NO")
	f.id, f.sku, f.upd = text(true), text(true), text(true)
	f.name = text(false)
	f.cat = iup.List()
	set(f.cat, "DROPDOWN", "YES", "VISIBLEITEMS", "8", "EXPAND", "HORIZONTAL")
	for i, c := range core.AllCategories {
		iup.SetAttributeId(f.cat, "", i+1, c.String())
	}
	f.price = text(false)
	f.qty = text(false)
	set(f.qty, "SPIN", "YES", "SPINMIN", "0", "SPINMAX", "1000000", "SPININC", "1", "SPINAUTO", "YES", "MASK", "/d*")
	for i, s := range core.AllStatuses {
		f.status[i] = iup.Toggle(s.String())
	}
	// The BackgroundBox gives the radio its own native parent view; see
	// radioGroup.
	statusRow := radioGroup(f.status[0], f.status[1], f.status[2])
	f.rating = iup.Val("HORIZONTAL")
	set(f.rating, "MIN", "0", "MAX", "5", "STEP", "0.02", "PAGESTEP", "0.1", "VALUE", "0", "EXPAND", "HORIZONTAL")
	f.ratingLbl = iup.Label("0.0")
	set(f.ratingLbl, "RASTERSIZE", "36x")
	f.featured = iup.Toggle("Featured")
	f.date = iup.DatePick()
	set(f.date, "ORDER", "YMD", "SEPARATOR", "-", "ZEROPRECED", "YES")
	f.dateLbl = iup.Label("none")
	set(f.dateLbl, "RASTERSIZE", "40x")
	f.dateClear = iup.Button("Clear")
	f.notes = iup.MultiLine()
	set(f.notes, "WORDWRAP", "YES", "VISIBLELINES", "4", "VISIBLECOLUMNS", "20", "EXPAND", "HORIZONTAL", "SCROLLBAR", "VERTICAL")

	f.save = iup.Button("Save")
	set(f.save, "SHOWASDEFAULT", "YES")
	f.revert = iup.Button("Revert")
	f.copyRow = iup.Button("Copy Row")
	f.importBtn = iup.Button("Import Notes…")
	f.dirtyLbl = iup.Label("")
	set(f.dirtyLbl, "RASTERSIZE", "120x")

	ratingRow := iup.Hbox(f.rating, f.ratingLbl)
	set(ratingRow, "ALIGNMENT", "ACENTER", "GAP", "4")
	dateRow := iup.Hbox(f.date, f.dateLbl, f.dateClear)
	set(dateRow, "ALIGNMENT", "ACENTER", "GAP", "6")
	buttons := iup.Hbox(f.save, f.revert, f.copyRow, f.importBtn)
	set(buttons, "GAP", "6")
	btnRow := iup.Vbox(buttons, f.dirtyLbl)
	set(btnRow, "GAP", "4")

	f.fields = []iup.Ihandle{f.name, f.cat, f.price, f.qty, f.featured, f.date, f.dateClear, f.notes,
		f.status[0], f.status[1], f.status[2], f.rating}

	body := iup.Vbox(
		row("ID", f.id), row("SKU", f.sku),
		row("Name", f.name), errRow("name"),
		row("Category", f.cat),
		row("Price", f.price), errRow("priceCents"),
		row("Quantity", f.qty), errRow("quantity"),
		row("Status", statusRow),
		row("Rating", ratingRow), errRow("rating"),
		row("Featured", f.featured),
		row("Restock date", dateRow), errRow("restockDate"),
		row("Updated", f.upd),
		row("Notes", f.notes), errRow("notes"),
	)
	set(body, "GAP", "6")
	f.box = iup.Vbox(f.hint, body, btnRow)
	set(f.box, "MARGIN", "8x4", "GAP", "8")
	f.hintFloating(false)

	// Callbacks. Programmatic VALUE changes do not call VALUECHANGED_CB, and
	// loading guards the toggles and lists, which do.
	changed := func(iup.Ihandle) int { a.formEdited(); return iup.DEFAULT }
	for _, t := range []iup.Ihandle{f.name, f.price, f.qty, f.notes} {
		iup.On(t, iup.ValueChangedCB, changed)
	}
	iup.On(f.cat, iup.ListActionCB, func(ih iup.Ihandle, text string, item, state int) int {
		if state == 1 {
			a.formEdited()
		}
		return iup.DEFAULT
	})
	for _, t := range f.status {
		iup.On(t, iup.ToggleActionCB, func(ih iup.Ihandle, st int) int {
			if st == 1 {
				a.formEdited()
			}
			return iup.DEFAULT
		})
	}
	iup.On(f.featured, iup.ToggleActionCB, func(ih iup.Ihandle, st int) int { a.formEdited(); return iup.DEFAULT })
	iup.On(f.rating, iup.ValueChangedCB, func(ih iup.Ihandle) int { a.formEdited(); return iup.DEFAULT })
	iup.On(f.date, iup.ValueChangedCB, func(ih iup.Ihandle) int {
		if !f.loading {
			a.f.cur.hasDate = true
		}
		a.formEdited()
		return iup.DEFAULT
	})
	iup.On(f.dateClear, iup.ActionCB, func(iup.Ihandle) int {
		a.f.cur.hasDate = false
		a.formEdited()
		return iup.DEFAULT
	})
	iup.On(f.save, iup.ActionCB, func(iup.Ihandle) int { a.save(); return iup.DEFAULT })
	iup.On(f.revert, iup.ActionCB, func(iup.Ihandle) int { a.revert(); return iup.DEFAULT })
	iup.On(f.copyRow, iup.ActionCB, func(iup.Ihandle) int { a.copyRow(); return iup.DEFAULT })
	iup.On(f.importBtn, iup.ActionCB, func(iup.Ihandle) int { a.beginImport(); return iup.DEFAULT })
	return f.box
}

func (f *formView) hintFloating(show bool) {
	// The label always takes part in the layout, so its natural size is
	// never lost and a later show cannot come back clipped. Hiding it only
	// clears VISIBLE (the row keeps its height).
	if show {
		set(f.hint, "VISIBLE", "YES")
	} else {
		set(f.hint, "VISIBLE", "NO")
	}
}

// formEdited reads the controls into the model after a user edit.
func (a *app) formEdited() {
	f := &a.ui.frm
	if f.loading || !a.f.has {
		return
	}
	a.readForm()
	a.uiFormState()
}

// readForm copies the control values into the form model.
func (a *app) readForm() {
	f := &a.ui.frm
	c := &a.f.cur
	c.name = iup.GetAttribute(f.name, "VALUE")
	c.price = iup.GetAttribute(f.price, "VALUE")
	c.qty = iup.GetAttribute(f.qty, "VALUE")
	c.notes = iup.GetAttribute(f.notes, "VALUE")
	if i, err := strconv.Atoi(iup.GetAttribute(f.cat, "VALUE")); err == nil && i >= 1 && i <= len(core.AllCategories) {
		c.cat = core.AllCategories[i-1].String()
	}
	for i, t := range f.status {
		if iup.GetAttribute(t, "VALUE") == "ON" {
			c.status = core.AllStatuses[i]
		}
	}
	if v, err := strconv.ParseFloat(strings.Replace(iup.GetAttribute(f.rating, "VALUE"), ",", ".", 1), 64); err == nil {
		c.rating = round1(v)
	}
	c.featured = iup.GetAttribute(f.featured, "VALUE") == "ON"
	if c.hasDate {
		if d, ok := parseDatePick(iup.GetAttribute(f.date, "VALUE")); ok {
			c.date = d
		}
	}
	set(f.ratingLbl, "TITLE", core.FormatRating(float32(c.rating)))
}

func parseDatePick(s string) (core.Date, bool) {
	var y, m, d int
	if _, err := fmt.Sscanf(strings.ReplaceAll(s, "-", "/"), "%d/%d/%d", &y, &m, &d); err != nil {
		return core.Date{}, false
	}
	return core.Date{Year: y, Month: monthOf(m), Day: d}, true
}

// uiForm shows the model in the controls.
func (a *app) uiForm() {
	f := &a.ui.frm
	if f.box == 0 {
		return
	}
	f.loading = true
	defer func() { f.loading = false }()
	has := a.f.has
	en := "NO"
	if has {
		en = "YES"
	}
	for _, w := range f.fields {
		set(w, "ACTIVE", en)
	}
	f.hintFloating(!has)
	if !has {
		for _, w := range []iup.Ihandle{f.id, f.sku, f.name, f.price, f.qty, f.upd, f.notes} {
			set(w, "VALUE", "")
		}
		set(f.ratingLbl, "TITLE", "")
		set(f.dateLbl, "TITLE", "")
	} else {
		it, c := a.f.item, a.f.cur
		set(f.id, "VALUE", strconv.FormatInt(it.ID, 10))
		set(f.sku, "VALUE", it.SKU)
		set(f.upd, "VALUE", core.FormatUpdated(it.UpdatedAt))
		set(f.name, "VALUE", c.name)
		set(f.price, "VALUE", c.price)
		set(f.qty, "VALUE", c.qty)
		set(f.notes, "VALUE", c.notes)
		for i, cat := range core.AllCategories {
			if cat.String() == c.cat {
				iup.SetAttribute(f.cat, "VALUE", i+1)
			}
		}
		for i, s := range core.AllStatuses {
			v := "OFF"
			if s == c.status {
				v = "ON"
			}
			set(f.status[i], "VALUE", v)
		}
		set(f.rating, "VALUE", fmt.Sprintf("%.1f", c.rating))
		set(f.ratingLbl, "TITLE", core.FormatRating(float32(c.rating)))
		v := "OFF"
		if c.featured {
			v = "ON"
		}
		set(f.featured, "VALUE", v)
		if c.hasDate {
			set(f.date, "VALUE", fmt.Sprintf("%d/%d/%d", c.date.Year, int(c.date.Month), c.date.Day))
			set(f.dateLbl, "TITLE", c.date.String())
		} else {
			set(f.dateLbl, "TITLE", "none")
		}
	}
	a.uiFormState()
}

// uiFormState updates what depends on the dirty state and the errors.
func (a *app) uiFormState() {
	f := &a.ui.frm
	dirty := a.f.dirty()
	en := "NO"
	if dirty {
		en = "YES"
	}
	// Both buttons are driven from the same dirty flag; Update repaints the
	// native button so the enabled state is shown at once.
	for _, b := range []iup.Ihandle{f.save, f.revert} {
		set(b, "ACTIVE", en)
		iup.Update(b)
	}
	hasItem := "NO"
	if a.f.has {
		hasItem = "YES"
	}
	set(f.copyRow, "ACTIVE", hasItem)
	set(f.importBtn, "ACTIVE", hasItem)
	if dirty {
		set(f.dirtyLbl, "TITLE", "Unsaved changes")
	} else {
		set(f.dirtyLbl, "TITLE", "")
	}
	if a.f.has {
		set(f.dateLbl, "TITLE", dateText(a.f.cur))
	}
	changed := false
	red := hexRGB(a.pal.StockFillLow)
	for field, row := range f.errs {
		msg := a.f.errs[field]
		el := f.errLbl[field]
		if msg != "" {
			if iup.GetAttribute(el, "TITLE") != msg || iup.GetAttribute(row, "FLOATING") != "NO" {
				set(el, "TITLE", msg, "FGCOLOR", red)
				set(row, "FLOATING", "NO", "VISIBLE", "YES")
				changed = true
			}
		} else if iup.GetAttribute(row, "FLOATING") != "IGNORE" {
			set(el, "TITLE", "")
			set(row, "FLOATING", "IGNORE", "VISIBLE", "NO")
			changed = true
		}
	}
	if changed {
		iup.Refresh(f.box)
	}
}

func dateText(c formFields) string {
	if c.hasDate {
		return c.date.String()
	}
	return "none"
}

func monthOf(m int) time.Month { return time.Month(m) }
