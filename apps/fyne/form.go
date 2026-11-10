package main

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"gguix/core"
)

// detailForm is the edit form for the selected item.
type detailForm struct {
	u *ui

	item    core.Item // the loaded item
	has     bool
	loading bool

	hint     *widget.Label
	id, sku  *widget.Label
	name     *widget.Entry
	category *widget.Select
	price    *widget.Entry
	qty      *widget.Entry
	qtyMinus *widget.Button
	qtyPlus  *widget.Button
	status   *widget.RadioGroup
	rating   *widget.Slider
	ratingLb *widget.Label
	featured *widget.Check
	restock  *widget.DateEntry
	clearDt  *widget.Button
	updated  *widget.Label
	notes    *widget.Entry

	errs map[string]*widget.Label

	save, revert, copyRow, importNotes *widget.Button
	dirtyLb                            *widget.Label

	root fyne.CanvasObject
}

var intRe = regexp.MustCompile(`^[0-9]+$`)

func newDetailForm(u *ui) *detailForm {
	f := &detailForm{u: u, errs: map[string]*widget.Label{}}
	change := func() { f.changed() }

	f.hint = widget.NewLabel("Select an item to edit.")
	f.id = widget.NewLabel("")
	f.sku = widget.NewLabel("")
	f.name = widget.NewEntry()
	f.name.OnChanged = func(string) { change() }
	var cats []string
	for _, c := range core.AllCategories {
		cats = append(cats, c.String())
	}
	f.category = widget.NewSelect(cats, func(string) { change() })
	f.price = widget.NewEntry()
	f.price.OnChanged = func(string) { change() }
	f.qty = widget.NewEntry()
	f.qty.Validator = func(s string) error {
		if !intRe.MatchString(strings.TrimSpace(s)) {
			return errInteger
		}
		return nil
	}
	f.qty.OnChanged = func(string) { change() }
	f.qtyMinus = widget.NewButton("-", func() { f.stepQty(-1) })
	f.qtyPlus = widget.NewButton("+", func() { f.stepQty(1) })
	var sts []string
	for _, s := range core.AllStatuses {
		sts = append(sts, s.String())
	}
	f.status = widget.NewRadioGroup(sts, func(string) { change() })
	f.status.Horizontal = true
	f.status.Required = true
	f.rating = widget.NewSlider(0, 5)
	f.rating.Step = 0.1
	f.ratingLb = widget.NewLabel("0.0")
	f.rating.OnChanged = func(v float64) {
		f.ratingLb.SetText(core.FormatRating(float32(v)))
		change()
	}
	f.featured = widget.NewCheck("", func(bool) { change() })
	f.restock = widget.NewDateEntry()
	f.restock.OnChanged = func(*time.Time) { change() }
	f.clearDt = widget.NewButton("Clear", func() {
		f.restock.SetDate(nil)
		change()
	})
	f.updated = widget.NewLabel("")
	f.notes = widget.NewMultiLineEntry()
	f.notes.Wrapping = fyne.TextWrapWord
	f.notes.SetMinRowsVisible(4)
	f.notes.OnChanged = func(string) { change() }

	f.save = widget.NewButton("Save", func() { f.doSave() })
	f.save.Importance = widget.HighImportance
	f.revert = widget.NewButton("Revert", func() { f.doRevert() })
	f.copyRow = widget.NewButton("Copy Row", func() { u.copyRow() })
	f.importNotes = widget.NewButton("Import Notes…", func() { u.importNotes() })
	f.dirtyLb = widget.NewLabel("Unsaved changes")
	f.dirtyLb.Importance = widget.WarningImportance
	f.dirtyLb.Hide()

	field := func(key string, obj fyne.CanvasObject) fyne.CanvasObject {
		e := widget.NewLabel("")
		e.Importance = widget.DangerImportance
		e.Wrapping = fyne.TextWrapWord
		e.Hide()
		f.errs[key] = e
		return container.NewVBox(obj, e)
	}
	lb := func(s string) fyne.CanvasObject { return widget.NewLabel(s) }

	qtyRow := container.NewBorder(nil, nil, nil, container.NewHBox(f.qtyMinus, f.qtyPlus), f.qty)
	ratingRow := container.NewBorder(nil, nil, nil, f.ratingLb, f.rating)
	restockRow := container.NewBorder(nil, nil, nil, f.clearDt, f.restock)

	form := container.New(layout.NewFormLayout(),
		lb("ID"), f.id,
		lb("SKU"), f.sku,
		lb("Name"), field("name", f.name),
		lb("Category"), field("category", f.category),
		lb("Price"), field("priceCents", f.price),
		lb("Quantity"), field("quantity", qtyRow),
		lb("Status"), field("status", f.status),
		lb("Rating"), field("rating", ratingRow),
		lb("Featured"), f.featured,
		lb("Restock date"), field("restockDate", restockRow),
		lb("Updated"), f.updated,
		lb("Notes"), field("notes", f.notes),
	)
	buttons := container.NewHBox(f.save, f.revert, f.copyRow, f.importNotes)
	f.root = container.NewVScroll(container.NewVBox(f.hint, form, buttons, f.dirtyLb))
	f.setEnabled(false)
	return f
}

type formError string

func (e formError) Error() string { return string(e) }

const errInteger = formError("Enter a whole number.")

func (f *detailForm) controls() []fyne.Disableable {
	return []fyne.Disableable{f.name, f.category, f.price, f.qty, f.qtyMinus, f.qtyPlus, f.status,
		f.rating, f.featured, f.restock, f.clearDt, f.notes, f.copyRow, f.importNotes}
}

func (f *detailForm) setEnabled(on bool) {
	for _, c := range f.controls() {
		if on {
			c.Enable()
		} else {
			c.Disable()
		}
	}
	if on {
		f.hint.Hide()
	} else {
		f.hint.Show()
		f.save.Disable()
		f.revert.Disable()
	}
}

func (f *detailForm) stepQty(d int) {
	n, err := strconv.Atoi(strings.TrimSpace(f.qty.Text))
	if err != nil {
		n = 0
	}
	n += d
	n = max(0, min(n, core.MaxQuantity))
	f.qty.SetText(strconv.Itoa(n))
}

// Load shows it in the form and clears all errors and the dirty state.
func (f *detailForm) Load(it core.Item) {
	f.loading = true
	f.item, f.has = it, true
	f.id.SetText(strconv.FormatInt(it.ID, 10))
	f.sku.SetText(it.SKU)
	f.name.SetText(it.Name)
	f.category.SetSelected(it.Category.String())
	f.price.SetText(core.FormatPrice(it.PriceCents))
	f.qty.SetText(strconv.Itoa(int(it.Quantity)))
	f.status.SetSelected(it.Status.String())
	f.rating.SetValue(float64(it.Rating))
	f.ratingLb.SetText(core.FormatRating(it.Rating))
	f.featured.SetChecked(it.Featured)
	if it.RestockDate != nil {
		t := it.RestockDate.Time()
		f.restock.SetDate(&t)
	} else {
		f.restock.SetDate(nil)
	}
	f.updated.SetText(core.FormatUpdated(it.UpdatedAt))
	f.notes.SetText(it.Notes)
	f.loading = false
	f.clearErrors()
	f.setEnabled(true)
	f.refreshDirty()
}

// Clear empties the form and disables it.
func (f *detailForm) Clear() {
	f.loading = true
	f.has = false
	f.item = core.Item{}
	f.id.SetText("")
	f.sku.SetText("")
	f.name.SetText("")
	f.category.ClearSelected()
	f.price.SetText("")
	f.qty.SetText("")
	f.status.SetSelected("")
	f.rating.SetValue(0)
	f.ratingLb.SetText("0.0")
	f.featured.SetChecked(false)
	f.restock.SetDate(nil)
	f.updated.SetText("")
	f.notes.SetText("")
	f.loading = false
	f.clearErrors()
	f.setEnabled(false)
	f.refreshDirty()
}

func (f *detailForm) clearErrors() {
	for _, e := range f.errs {
		e.SetText("")
		e.Hide()
	}
}

func (f *detailForm) showErrors(errs []core.FieldError) {
	f.clearErrors()
	for _, fe := range errs {
		if l, ok := f.errs[fe.Field]; ok {
			if l.Text != "" {
				l.SetText(l.Text + "\n" + fe.Message)
			} else {
				l.SetText(fe.Message)
			}
			l.Show()
		}
	}
}

// Errors returns the messages currently shown, by field.
func (f *detailForm) Errors() map[string]string {
	out := map[string]string{}
	for k, l := range f.errs {
		if l.Visible() && l.Text != "" {
			out[k] = l.Text
		}
	}
	return out
}

// read builds the item described by the form. Parse problems are returned as
// field errors; fields that cannot be parsed keep the loaded value.
func (f *detailForm) read() (core.Item, []core.FieldError) {
	it := f.item
	var errs []core.FieldError
	it.Name = f.name.Text
	if c, err := core.ParseCategory(f.category.Selected); err == nil {
		it.Category = c
	}
	if cents, err := core.ParsePrice(f.price.Text); err == nil {
		it.PriceCents = cents
	} else {
		errs = append(errs, core.FieldError{Field: "priceCents", Message: err.Error()})
	}
	if n, err := strconv.ParseInt(strings.TrimSpace(f.qty.Text), 10, 64); err == nil && n >= 0 && n <= math.MaxInt32 {
		it.Quantity = int32(n)
	} else if err == nil {
		errs = append(errs, core.FieldError{Field: "quantity", Message: core.MsgQuantity})
	} else {
		errs = append(errs, core.FieldError{Field: "quantity", Message: errInteger.Error()})
	}
	if s, err := core.ParseStatus(f.status.Selected); err == nil {
		it.Status = s
	}
	it.Rating = core.RatingFromTenths(int(math.Round(f.rating.Value * 10)))
	it.Featured = f.featured.Checked
	if f.restock.Date != nil {
		d := core.DateOf(*f.restock.Date)
		it.RestockDate = &d
	} else {
		it.RestockDate = nil
	}
	it.Notes = f.notes.Text
	return it, errs
}

func sameDate(a, b *core.Date) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// dirty reports whether the form differs from the loaded item.
func (f *detailForm) dirty() bool {
	if !f.has {
		return false
	}
	it, errs := f.read()
	if len(errs) > 0 {
		return true
	}
	o := f.item
	return it.Name != o.Name || it.Category != o.Category || it.PriceCents != o.PriceCents ||
		it.Quantity != o.Quantity || it.Status != o.Status || it.Rating != o.Rating ||
		it.Featured != o.Featured || !sameDate(it.RestockDate, o.RestockDate) || it.Notes != o.Notes
}

func (f *detailForm) changed() {
	if f.loading {
		return
	}
	f.refreshDirty()
}

func (f *detailForm) refreshDirty() {
	d := f.dirty()
	if d {
		f.dirtyLb.Show()
		f.save.Enable()
		f.revert.Enable()
	} else {
		f.dirtyLb.Hide()
		f.save.Disable()
		f.revert.Disable()
	}
	f.u.updateMenus()
}

func (f *detailForm) doRevert() {
	if !f.has || !f.dirty() {
		return
	}
	f.u.confirmDiscard(func() { f.Load(f.item) })
}

func (f *detailForm) doSave() {
	if !f.has {
		return
	}
	it, perrs := f.read()
	if len(perrs) > 0 {
		f.showErrors(perrs)
		return
	}
	if verr := core.ValidateItem(it); verr != nil {
		f.showErrors(verr.Errors)
		return
	}
	f.save.Disable()
	f.u.saveItem(it, func(saved core.Item, err error) {
		if err != nil {
			if verr, ok := err.(*core.ValidationError); ok {
				f.showErrors(verr.Errors)
			} else {
				f.u.showError(err)
			}
			f.refreshDirty()
			return
		}
		f.Load(saved)
	})
}

// SetNotes puts imported text into the notes field (the form becomes dirty).
func (f *detailForm) SetNotes(s string) {
	f.notes.SetText(s)
}
