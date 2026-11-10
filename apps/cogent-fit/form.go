package main

// The detail form (brief section 5.4), shared by the desktop and web targets.

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"

	gc "gguix/core"
)

// detailForm is the editor for the selected item.
type detailForm struct {
	app  *App
	root *core.Frame

	msg                  *core.Text
	idT, skuT, updT      *core.Text
	name                 *core.TextField
	cat                  *core.Chooser
	price                *core.TextField
	qty                  *core.Spinner
	status               *core.Switches
	rating               *core.Slider
	ratingT              *core.Text
	featured             *core.Switch
	restock              *core.TextField
	restockClear         *core.Button
	notes                *textcore.Editor
	errs                 map[string]*core.Text
	saveB, revertB       *core.Button
	copyB, importB       *core.Button
	dirtyT               *core.Text
	controls             []core.Widget
	loaded               gc.Item
	has, saving, loading bool
	updateCalls          int // for tests
	errMsgs              map[string]string
}

var statusNames = []string{"Active", "Backordered", "Discontinued"}

func newDetailForm(a *App, parent core.Widget) *detailForm {
	f := &detailForm{app: a, errs: map[string]*core.Text{}, errMsgs: map[string]string{}}
	f.root = core.NewFrame(parent)
	f.root.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.Overflow.Y = styles.OverflowAuto
		s.Gap.Set(units.Dp(6))
		s.Min.X.Dp(280)
	})
	f.msg = core.NewText(f.root).SetType(core.TextSupporting).SetText("Select an item to edit.")

	grid := core.NewFrame(f.root)
	grid.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Dp(8), units.Dp(6))
	})
	label := func(t string) {
		l := core.NewText(grid).SetType(core.TextLabelLarge).SetText(t)
		l.Styler(func(s *styles.Style) { s.Align.Self = styles.Center; s.Min.X.Dp(80) })
	}
	cell := func(field string) *core.Frame {
		c := core.NewFrame(grid)
		c.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
			s.Gap.Set(units.Dp(2))
			s.Padding.Zero()
		})
		return c
	}
	errText := func(c *core.Frame, field string) {
		e := core.NewText(c).SetType(core.TextBodySmall)
		e.Styler(func(s *styles.Style) {
			s.Color = colors.Scheme.Error.Base
			if f.errMsgs[field] == "" {
				s.Display = styles.DisplayNone
			}
		})
		e.Updater(func() { e.SetText(f.errMsgs[field]) })
		f.errs[field] = e
	}
	ctl := func(w core.Widget) { f.controls = append(f.controls, w) }
	grow := func(w core.Widget) {
		w.AsWidget().Styler(func(s *styles.Style) { s.Grow.Set(1, 0) })
	}

	label("ID")
	f.idT = core.NewText(grid).SetType(core.TextBodyLarge)
	label("SKU")
	f.skuT = core.NewText(grid).SetType(core.TextBodyLarge)

	label("Name")
	c := cell("name")
	f.name = core.NewTextField(c)
	grow(f.name)
	f.name.OnInput(func(e events.Event) { f.changed() })
	ctl(f.name)
	errText(c, "name")

	label("Category")
	c = cell("category")
	f.cat = core.NewChooser(c)
	for _, cat := range gc.AllCategories {
		f.cat.Items = append(f.cat.Items, core.ChooserItem{Value: cat.String()})
	}
	f.cat.OnChange(func(e events.Event) { f.changed() })
	ctl(f.cat)

	label("Price")
	c = cell("priceCents")
	f.price = core.NewTextField(c)
	grow(f.price)
	f.price.OnInput(func(e events.Event) { f.changed() })
	ctl(f.price)
	errText(c, "priceCents")

	label("Quantity")
	c = cell("quantity")
	f.qty = core.NewSpinner(c)
	f.qty.SetMin(0).SetMax(1_000_000).SetStep(1).SetEnforceStep(true).SetFormat("%.0f")
	f.qty.OnChange(func(e events.Event) { f.changed() })
	f.qty.OnInput(func(e events.Event) { f.changed() })
	ctl(f.qty)
	errText(c, "quantity")

	label("Status")
	c = cell("status")
	f.status = core.NewSwitches(c).SetType(core.SwitchRadioButton).SetMutex(true).SetStrings(statusNames...)
	f.status.OnChange(func(e events.Event) { f.changed() })
	ctl(f.status)

	label("Rating")
	c = cell("rating")
	rr := core.NewFrame(c)
	rr.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Dp(8))
		s.Align.Items = styles.Center
		s.Padding.Zero()
	})
	f.rating = core.NewSlider(rr).SetMin(0).SetMax(5).SetStep(0.1).SetEnforceStep(true)
	grow(f.rating)
	f.rating.OnInput(func(e events.Event) { f.ratingMoved() })
	f.rating.OnChange(func(e events.Event) { f.ratingMoved() })
	ctl(f.rating)
	f.ratingT = core.NewText(rr).SetType(core.TextBodyLarge)
	f.ratingT.Styler(func(s *styles.Style) { s.Min.X.Ch(3); s.Text.Align = text.End })
	errText(c, "rating")

	label("Featured")
	f.featured = core.NewSwitch(grid).SetType(core.SwitchCheckbox)
	f.featured.OnChange(func(e events.Event) { f.changed() })
	ctl(f.featured)

	label("Restock date")
	c = cell("restockDate")
	rd := core.NewFrame(c)
	rd.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Gap.Set(units.Dp(6))
		s.Align.Items = styles.Center
		s.Padding.Zero()
	})
	f.restock = core.NewTextField(rd).SetPlaceholder("YYYY-MM-DD")
	grow(f.restock)
	f.restock.SetLeadingIcon("calendar_today", func(e events.Event) { f.pickDate() })
	f.restock.OnInput(func(e events.Event) { f.changed() })
	ctl(f.restock)
	f.restockClear = core.NewButton(rd).SetText("Clear")
	f.restockClear.OnClick(func(e events.Event) {
		f.restock.SetText("")
		f.restock.NeedsRender()
		f.changed()
	})
	ctl(f.restockClear)
	errText(c, "restockDate")

	label("Updated")
	f.updT = core.NewText(grid).SetType(core.TextBodyLarge)

	label("Notes")
	c = cell("notes")
	f.notes = textcore.NewEditor(c)
	f.notes.Lines.SetString("")
	f.notes.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Min.Y.Em(6)
		s.Max.Y.Em(6)
		s.Text.WhiteSpace = text.WrapAsNeeded
	})
	f.notes.OnInput(func(e events.Event) { f.changed() })
	ctl(f.notes)
	errText(c, "notes")

	// buttons
	bar := core.NewFrame(f.root)
	bar.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Gap.Set(units.Dp(8))
		s.Align.Items = styles.Center
		s.Grow.Set(1, 0)
		s.Wrap = true
	})
	f.saveB = core.NewButton(bar).SetText("Save")
	f.saveB.OnClick(func(e events.Event) { f.save() })
	f.revertB = core.NewButton(bar).SetText("Revert").SetType(core.ButtonOutlined)
	f.revertB.OnClick(func(e events.Event) { f.revert() })
	f.copyB = core.NewButton(bar).SetText("Copy Row").SetType(core.ButtonOutlined)
	f.copyB.OnClick(func(e events.Event) { f.copyRow() })
	f.importB = core.NewButton(bar).SetText("Import Notes…").SetType(core.ButtonOutlined)
	f.importB.OnClick(func(e events.Event) { f.importNotes() })
	f.dirtyT = core.NewText(bar).SetType(core.TextBodyMedium).SetText("Unsaved changes")

	f.clear()
	return f
}

// ratingMoved updates the value label and dirty state.
func (f *detailForm) ratingMoved() {
	f.ratingT.SetText(gc.FormatRating(f.ratingValue()))
	f.ratingT.NeedsRender()
	f.changed()
}

func (f *detailForm) ratingValue() float32 {
	return gc.RatingFromTenths(int(math.Round(float64(f.rating.Value) * 10)))
}

func (f *detailForm) statusValue() gc.Status {
	if it := f.status.SelectedItem(); it != nil {
		if s, err := gc.ParseStatus(it.Value.(string)); err == nil {
			return s
		}
	}
	return f.loaded.Status
}

func (f *detailForm) notesText() string { return f.notes.Lines.String() }

// dirty reports whether the form differs from the loaded item.
func (f *detailForm) dirty() bool {
	if !f.has {
		return false
	}
	l := f.loaded
	return f.name.Text() != l.Name ||
		f.cat.CurrentIndex != int(l.Category) ||
		strings.TrimSpace(f.price.Text()) != gc.FormatPrice(l.PriceCents) ||
		int32(math.Round(float64(f.qty.Value))) != l.Quantity ||
		f.statusValue() != l.Status ||
		gc.RatingTenths(f.ratingValue()) != gc.RatingTenths(l.Rating) ||
		f.featured.IsChecked() != l.Featured ||
		strings.TrimSpace(f.restock.Text()) != restockString(l) ||
		strings.TrimRight(f.notesText(), "\n") != strings.TrimRight(l.Notes, "\n")
}

func restockString(it gc.Item) string {
	if it.RestockDate == nil {
		return ""
	}
	return it.RestockDate.String()
}

// changed is called whenever a control changes.
func (f *detailForm) changed() {
	if f.loading {
		return
	}
	f.updateButtons()
}

func (f *detailForm) updateButtons() {
	d := f.dirty()
	f.saveB.SetEnabled(f.has && d && !f.saving)
	f.revertB.SetEnabled(f.has && d && !f.saving)
	f.copyB.SetEnabled(f.has)
	f.importB.SetEnabled(f.has)
	f.dirtyT.SetText(map[bool]string{true: "Unsaved changes", false: ""}[d])
	f.dirtyT.NeedsRender()
	f.saveB.NeedsRender()
	f.revertB.NeedsRender()
}

// DirtyText returns the text shown next to the buttons (for tests).
func (f *detailForm) DirtyText() string { return f.dirtyT.Text }

// load shows an item.
func (f *detailForm) load(it gc.Item) {
	f.loading = true
	f.loaded = it
	f.has = true
	f.msg.SetText("")
	f.idT.SetText(strconv.FormatInt(it.ID, 10))
	f.skuT.SetText(it.SKU)
	f.name.SetText(it.Name)
	f.cat.SetCurrentIndex(int(it.Category))
	f.price.SetText(gc.FormatPrice(it.PriceCents))
	f.qty.SetValue(float32(it.Quantity))
	f.status.SelectValue(it.Status.String())
	f.rating.SetValue(it.Rating)
	f.ratingT.SetText(gc.FormatRating(it.Rating))
	f.featured.SetChecked(it.Featured)
	f.restock.SetText(restockString(it))
	f.updT.SetText(gc.FormatUpdated(it.UpdatedAt))
	f.notes.Lines.SetString(it.Notes)
	f.setErrors(nil)
	f.setEnabled(true)
	f.loading = false
	f.updateButtons()
	f.root.Update()
}

// clear shows the empty state.
func (f *detailForm) clear() {
	f.loading = true
	f.has = false
	f.loaded = gc.Item{}
	f.msg.SetText("Select an item to edit.")
	f.idT.SetText("")
	f.skuT.SetText("")
	f.name.SetText("")
	f.cat.SetCurrentIndex(0)
	f.price.SetText("")
	f.qty.SetValue(0)
	f.status.SelectValue("Active")
	f.rating.SetValue(0)
	f.ratingT.SetText(gc.FormatRating(0))
	f.featured.SetChecked(false)
	f.restock.SetText("")
	f.updT.SetText("")
	f.notes.Lines.SetString("")
	f.setErrors(nil)
	f.setEnabled(false)
	f.loading = false
	f.updateButtons()
	f.root.Update()
}

func (f *detailForm) setEnabled(on bool) {
	for _, w := range f.controls {
		w.AsWidget().SetEnabled(on)
	}
}

func (f *detailForm) setErrors(m map[string]string) {
	f.errMsgs = map[string]string{}
	for k, v := range m {
		f.errMsgs[k] = v
	}
	for k, e := range f.errs {
		e.SetText(f.errMsgs[k])
		e.Restyle()
	}
	f.root.NeedsLayout()
}

// Errors returns the visible field error messages (for tests).
func (f *detailForm) Errors() map[string]string { return f.errMsgs }

// collect reads the form into an item and returns field errors.
func (f *detailForm) collect() (gc.Item, map[string]string) {
	errs := map[string]string{}
	it := f.loaded
	it.Name = strings.TrimSpace(f.name.Text())
	it.Category = gc.Category(f.cat.CurrentIndex)
	priceOK := true
	if cents, err := gc.ParsePrice(f.price.Text()); err != nil {
		errs["priceCents"] = err.Error()
		priceOK = false
	} else {
		it.PriceCents = cents
	}
	it.Quantity = int32(math.Round(float64(f.qty.Value)))
	it.Status = f.statusValue()
	it.Rating = f.ratingValue()
	it.Featured = f.featured.IsChecked()
	rs := strings.TrimSpace(f.restock.Text())
	dateOK := true
	if rs == "" {
		it.RestockDate = nil
	} else if d, err := gc.ParseDate(rs); err != nil {
		errs["restockDate"] = "Enter a date like 2026-03-07."
		dateOK = false
	} else {
		it.RestockDate = &d
	}
	it.Notes = strings.TrimRight(f.notesText(), "\n")
	if ve := gc.ValidateItem(it); ve != nil {
		for _, fe := range ve.Errors {
			if fe.Field == "priceCents" && !priceOK {
				continue
			}
			if fe.Field == "restockDate" && !dateOK {
				continue
			}
			if _, dup := errs[fe.Field]; !dup {
				errs[fe.Field] = fe.Message
			}
		}
	}
	return it, errs
}

// save validates and saves the item.
func (f *detailForm) save() {
	if !f.has || f.saving {
		return
	}
	it, errs := f.collect()
	f.setErrors(errs)
	if len(errs) > 0 {
		return
	}
	f.saving = true
	f.updateButtons()
	svc := f.app.svc
	f.updateCalls++
	go func() {
		saved, err := svc.Update(context.Background(), it)
		f.app.async(func() {
			f.saving = false
			if err != nil {
				var ve *gc.ValidationError
				if errors.As(err, &ve) {
					m := map[string]string{}
					for _, fe := range ve.Errors {
						if _, dup := m[fe.Field]; !dup {
							m[fe.Field] = fe.Message
						}
					}
					f.setErrors(m)
				} else {
					core.ErrorDialog(f.app.body, err, "Save failed")
				}
				f.updateButtons()
				return
			}
			f.load(saved)
			f.app.afterSave(saved)
		})
	}()
}

func (f *detailForm) revert() {
	if !f.has {
		return
	}
	f.app.confirmDiscard(func() {
		f.load(f.loaded)
	})
}

func (f *detailForm) copyRow() {
	if !f.has {
		return
	}
	f.root.Clipboard().Write(mimedata.NewText(gc.ItemCSVRow(f.loaded)))
	f.app.flash("Row copied")
}

// pickDate opens the framework date picker.
func (f *detailForm) pickDate() {
	start := time.Now()
	if d, err := gc.ParseDate(strings.TrimSpace(f.restock.Text())); err == nil {
		start = time.Date(d.Year, d.Month, d.Day, 12, 0, 0, 0, time.Local)
	}
	d := core.NewBody("Restock date")
	dp := core.NewDatePicker(d).SetTime(start)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			gd := gc.Date{Year: dp.Time.Year(), Month: dp.Time.Month(), Day: dp.Time.Day()}
			f.restock.SetText(gd.String())
			f.restock.NeedsRender()
			f.changed()
		})
	})
	d.RunDialog(f.restock)
}

const maxImportBytes = 64 * 1024

// applyImported puts imported text into the notes field.
func (f *detailForm) applyImported(data []byte) {
	if len(data) > maxImportBytes {
		core.ErrorDialog(f.app.body, errors.New("File is too large."), "Import Notes")
		return
	}
	s := string(data)
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	r := []rune(s)
	if len(r) > 500 {
		r = r[:500]
	}
	f.notes.Lines.SetString(string(r))
	f.notes.NeedsRender()
	f.changed()
}

var (
	_ = lines.NewLines
	_ = tree.Continue
)
