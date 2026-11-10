//go:build !js

package main

import (
	"os"
	"strings"
	"unicode/utf8"

	"gguix/core"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
)

// formState is the editable content of the form, used for dirty checks.
type formState struct {
	Name     string
	Category int
	Price    string
	Qty      int
	Status   int
	Rating   int // tenths
	Featured bool
	Restock  string // YYYY-MM-DD or ""
	Notes    string
}

func stateOf(it core.Item) formState {
	s := formState{
		Name: it.Name, Category: int(it.Category), Price: core.FormatPrice(it.PriceCents),
		Qty: int(it.Quantity), Status: int(it.Status), Rating: core.RatingTenths(it.Rating),
		Featured: it.Featured, Notes: it.Notes,
	}
	if it.RestockDate != nil {
		s.Restock = it.RestockDate.String()
	}
	return s
}

// noDate is the date shown as "None": the minimum of the date edit.
var noDate = [3]int{1999, 12, 31}

type detailForm struct {
	a    *App
	root *qt.QWidget

	hint   *qt.QLabel
	fields *qt.QWidget
	idL    *qt.QLabel
	skuL   *qt.QLabel
	updL   *qt.QLabel

	name         *qt.QLineEdit
	cat          *qt.QComboBox
	price        *qt.QLineEdit
	qty          *qt.QSpinBox
	statusBtns   []*qt.QRadioButton
	statusGroup  *qt.QButtonGroup
	rating       *qt.QSlider
	ratingL      *qt.QLabel
	featured     *qt.QCheckBox
	restock      *qt.QDateEdit
	restockClear *qt.QPushButton
	notes        *qt.QPlainTextEdit
	errs         map[string]*qt.QLabel

	save, revert, copyRow, importBtn *qt.QPushButton
	dirtyL                           *qt.QLabel

	item    *core.Item
	base    formState
	loading bool
	saving  bool
	dirty   bool
}

func newDetailForm(a *App) *detailForm {
	f := &detailForm{a: a, errs: map[string]*qt.QLabel{}}
	f.root = qt.NewQWidget2()
	outer := qt.NewQVBoxLayout(f.root)
	f.hint = qt.NewQLabel3("Select an item to edit.")
	outer.AddWidget(f.hint.QWidget)

	f.fields = qt.NewQWidget2()
	form := qt.NewQFormLayout(f.fields)
	form.SetLabelAlignment(qt.AlignRight | qt.AlignTop)

	f.idL = qt.NewQLabel2()
	f.skuL = qt.NewQLabel2()
	f.updL = qt.NewQLabel2()
	for _, l := range []*qt.QLabel{f.idL, f.skuL, f.updL} {
		l.SetTextInteractionFlags(qt.TextSelectableByMouse)
	}

	f.name = qt.NewQLineEdit2()
	f.cat = qt.NewQComboBox2()
	for _, c := range core.AllCategories {
		f.cat.AddItem(c.String())
	}
	f.price = qt.NewQLineEdit2()
	f.qty = qt.NewQSpinBox2()
	f.qty.SetRange(0, 1_000_000)
	f.qty.SetSingleStep(1)

	statusRow := qt.NewQWidget2()
	sh := qt.NewQHBoxLayout(statusRow)
	sh.SetContentsMargins(0, 0, 0, 0)
	f.statusGroup = qt.NewQButtonGroup2(statusRow.QObject)
	for i, s := range core.AllStatuses {
		b := qt.NewQRadioButton3(s.String())
		f.statusGroup.AddButton2(b.QAbstractButton, i)
		sh.AddWidget(b.QWidget)
		f.statusBtns = append(f.statusBtns, b)
	}
	sh.AddStretch()

	ratingRow := qt.NewQWidget2()
	rh := qt.NewQHBoxLayout(ratingRow)
	rh.SetContentsMargins(0, 0, 0, 0)
	f.rating = qt.NewQSlider3(qt.Horizontal)
	f.rating.SetRange(0, 50)
	f.rating.SetSingleStep(1)
	f.rating.SetPageStep(5)
	f.ratingL = qt.NewQLabel3(core.FormatRating(0))
	f.ratingL.SetMinimumWidth(f.ratingL.FontMetrics().HorizontalAdvance("5.0") + 4)
	rh.AddWidget2(f.rating.QWidget, 1)
	rh.AddWidget(f.ratingL.QWidget)

	f.featured = qt.NewQCheckBox2()
	restockRow := qt.NewQWidget2()
	dh := qt.NewQHBoxLayout(restockRow)
	dh.SetContentsMargins(0, 0, 0, 0)
	f.restock = qt.NewQDateEdit2()
	f.restock.SetCalendarPopup(true)
	f.restock.SetDisplayFormat("yyyy-MM-dd")
	f.restock.SetDateRange(*qt.NewQDate2(noDate[0], noDate[1], noDate[2]), *qt.NewQDate2(9999, 12, 31))
	f.restock.SetSpecialValueText("None")
	f.restockClear = qt.NewQPushButton3("Clear")
	dh.AddWidget2(f.restock.QWidget, 1)
	dh.AddWidget(f.restockClear.QWidget)

	f.notes = qt.NewQPlainTextEdit2()
	f.notes.SetFixedHeight(4*f.notes.FontMetrics().LineSpacing() + 2*int(f.notes.Document().DocumentMargin()) + 2*f.notes.FrameWidth() + 2)

	add := func(label, key string, w *qt.QWidget) {
		box := qt.NewQWidget2()
		v := qt.NewQVBoxLayout(box)
		v.SetContentsMargins(0, 0, 0, 0)
		v.SetSpacing(2)
		v.AddWidget(w)
		if key != "" {
			e := qt.NewQLabel2()
			e.SetWordWrap(true)
			e.SetVisible(false)
			f.styleError(e)
			f.errs[key] = e
			v.AddWidget(e.QWidget)
		}
		form.AddRow3(label, box)
	}
	add("ID", "", f.idL.QWidget)
	add("SKU", "", f.skuL.QWidget)
	add("Name", "name", f.name.QWidget)
	add("Category", "category", f.cat.QWidget)
	add("Price", "priceCents", f.price.QWidget)
	add("Quantity", "quantity", f.qty.QWidget)
	add("Status", "status", statusRow)
	add("Rating", "rating", ratingRow)
	add("Featured", "", f.featured.QWidget)
	add("Restock date", "restockDate", restockRow)
	add("Updated", "", f.updL.QWidget)
	add("Notes", "notes", f.notes.QWidget)
	outer.AddWidget(f.fields)

	btns := qt.NewQHBoxLayout2()
	f.save = qt.NewQPushButton3("Save")
	f.save.SetDefault(true)
	f.revert = qt.NewQPushButton3("Revert")
	f.copyRow = qt.NewQPushButton3("Copy Row")
	f.importBtn = qt.NewQPushButton3("Import Notes…")
	f.dirtyL = qt.NewQLabel3("Unsaved changes")
	f.dirtyL.SetVisible(false)
	for _, b := range []*qt.QPushButton{f.save, f.revert, f.copyRow, f.importBtn} {
		btns.AddWidget(b.QWidget)
	}
	btns.AddWidget(f.dirtyL.QWidget)
	btns.AddStretch()
	outer.AddLayout(btns.QLayout)
	outer.AddStretch()

	// change tracking
	touch := func() { f.onEdited() }
	f.name.OnTextChanged(func(string) { touch() })
	f.cat.OnCurrentIndexChanged(func(int) { touch() })
	f.price.OnTextChanged(func(string) { touch() })
	f.qty.OnValueChanged(func(int) { touch() })
	f.statusGroup.OnIdToggled(func(int, bool) { touch() })
	f.rating.OnValueChanged(func(v int) {
		f.ratingL.SetText(core.FormatRating(core.RatingFromTenths(v)))
		touch()
	})
	f.featured.OnToggled(func(bool) { touch() })
	f.restock.OnDateChanged(func(qt.QDate) { touch() })
	f.notes.OnTextChanged(touch)
	f.restockClear.OnClicked(func() { f.setRestock("") })

	f.save.OnClicked(f.doSave)
	f.revert.OnClicked(f.doRevert)
	f.copyRow.OnClicked(f.doCopyRow)
	f.importBtn.OnClicked(f.doImport)

	f.clear()
	return f
}

// errorColor is the color for validation messages. Qt Widgets has no
// semantic error color in QPalette, so this uses BrightText when the theme
// makes it red and otherwise the platform's standard system red.
func errorColor(dark bool) *qt.QColor {
	c := qt.QGuiApplication_Palette().BrightText().Color()
	if c.Red() > 150 && c.Green() < 110 && c.Blue() < 110 {
		return c
	}
	if dark {
		return qt.NewQColor3(255, 69, 58)
	}
	return qt.NewQColor3(215, 0, 21)
}

func (f *detailForm) styleError(l *qt.QLabel) {
	pal := qt.NewQPalette7(l.Palette())
	pal.SetColor2(qt.QPalette__WindowText, errorColor(f.a.isDark()))
	l.SetPalette(pal)
}

func (f *detailForm) restyleErrors() {
	for _, l := range f.errs {
		f.styleError(l)
	}
}

func (f *detailForm) controls() []*qt.QWidget {
	return []*qt.QWidget{f.fields, f.save.QWidget, f.revert.QWidget, f.copyRow.QWidget, f.importBtn.QWidget}
}

func (f *detailForm) clearErrors() {
	for _, l := range f.errs {
		l.SetText("")
		l.SetVisible(false)
	}
}

func (f *detailForm) showErrors(errs []core.FieldError) {
	f.clearErrors()
	for _, e := range errs {
		key := e.Field
		if l, ok := f.errs[key]; ok {
			t := e.Message
			if l.Text() != "" {
				t = l.Text() + "\n" + t
			}
			l.SetText(t)
			l.SetVisible(true)
		}
	}
}

// errorText returns the message shown under a field ("" if none).
func (f *detailForm) errorText(field string) string {
	if l, ok := f.errs[field]; ok && !l.IsHidden() {
		return l.Text()
	}
	return ""
}

func (f *detailForm) setRestock(s string) {
	if s == "" {
		f.restock.SetDate(*qt.NewQDate2(noDate[0], noDate[1], noDate[2]))
		return
	}
	d, err := core.ParseDate(s)
	if err != nil {
		f.restock.SetDate(*qt.NewQDate2(noDate[0], noDate[1], noDate[2]))
		return
	}
	f.restock.SetDate(*qt.NewQDate2(d.Year, int(d.Month), d.Day))
}

func (f *detailForm) restockString() string {
	d := f.restock.Date()
	if d.Year() == noDate[0] && d.Month() == noDate[1] && d.Day() == noDate[2] {
		return ""
	}
	return core.Date{Year: d.Year(), Month: timeMonth(d.Month()), Day: d.Day()}.String()
}

// load shows an item and makes it the clean baseline.
func (f *detailForm) load(it core.Item) {
	f.loading = true
	cp := it
	f.item = &cp
	f.idL.SetText(itoa64(it.ID))
	f.skuL.SetText(it.SKU)
	f.updL.SetText(core.FormatUpdated(it.UpdatedAt))
	s := stateOf(it)
	f.name.SetText(s.Name)
	f.cat.SetCurrentIndex(s.Category)
	f.price.SetText(s.Price)
	f.qty.SetValue(s.Qty)
	f.statusBtns[s.Status].SetChecked(true)
	f.rating.SetValue(s.Rating)
	f.ratingL.SetText(core.FormatRating(core.RatingFromTenths(s.Rating)))
	f.featured.SetChecked(s.Featured)
	f.setRestock(s.Restock)
	f.notes.SetPlainText(s.Notes)
	f.loading = false
	f.base = f.state()
	f.clearErrors()
	f.hint.SetVisible(false)
	f.setEnabled(true)
	f.updateDirty()
}

// clear shows the empty, disabled form.
func (f *detailForm) clear() {
	f.loading = true
	f.item = nil
	f.idL.SetText("")
	f.skuL.SetText("")
	f.updL.SetText("")
	f.name.SetText("")
	f.cat.SetCurrentIndex(0)
	f.price.SetText("")
	f.qty.SetValue(0)
	f.statusBtns[0].SetChecked(true)
	f.rating.SetValue(0)
	f.featured.SetChecked(false)
	f.setRestock("")
	f.notes.SetPlainText("")
	f.loading = false
	f.base = f.state()
	f.clearErrors()
	f.hint.SetVisible(true)
	f.setEnabled(false)
	f.updateDirty()
}

func (f *detailForm) setEnabled(on bool) {
	for _, w := range f.controls() {
		w.SetEnabled(on)
	}
}

func (f *detailForm) state() formState {
	return formState{
		Name: f.name.Text(), Category: f.cat.CurrentIndex(), Price: f.price.Text(),
		Qty: f.qty.Value(), Status: max(0, f.statusGroup.CheckedId()), Rating: f.rating.Value(),
		Featured: f.featured.IsChecked(), Restock: f.restockString(), Notes: f.notes.ToPlainText(),
	}
}

func (f *detailForm) isDirty() bool { return f.dirty }

func (f *detailForm) onEdited() {
	if f.loading {
		return
	}
	f.updateDirty()
}

func (f *detailForm) updateDirty() {
	f.dirty = f.item != nil && f.state() != f.base
	f.dirtyL.SetVisible(f.dirty)
	f.save.SetEnabled(f.dirty && !f.saving)
	f.revert.SetEnabled(f.dirty)
	if f.item == nil {
		f.save.SetEnabled(false)
		f.revert.SetEnabled(false)
	}
}

// draft builds the item described by the form, plus any parse errors.
func (f *detailForm) draft() (core.Item, []core.FieldError) {
	it := *f.item
	var errs []core.FieldError
	it.Name = strings.TrimSpace(f.name.Text())
	it.Category = core.Category(f.cat.CurrentIndex())
	priceOK := true
	if c, err := core.ParsePrice(f.price.Text()); err != nil {
		errs = append(errs, core.FieldError{Field: "priceCents", Message: err.Error()})
		priceOK = false
	} else {
		it.PriceCents = c
	}
	it.Quantity = int32(f.qty.Value())
	it.Status = core.Status(max(0, f.statusGroup.CheckedId()))
	it.Rating = core.RatingFromTenths(f.rating.Value())
	it.Featured = f.featured.IsChecked()
	it.RestockDate = nil
	if s := f.restockString(); s != "" {
		if d, err := core.ParseDate(s); err == nil {
			it.RestockDate = &d
		}
	}
	it.Notes = f.notes.ToPlainText()
	if ve := core.ValidateItem(it); ve != nil {
		for _, e := range ve.Errors {
			if e.Field == "priceCents" && !priceOK {
				continue
			}
			errs = append(errs, e)
		}
	}
	return it, errs
}

func (f *detailForm) doSave() {
	if f.item == nil || !f.dirty || f.saving {
		return
	}
	it, errs := f.draft()
	if len(errs) > 0 {
		f.showErrors(errs)
		return
	}
	f.clearErrors()
	f.saving = true
	f.updateDirty()
	svc, ctx := f.a.svc, f.a.ctx
	id := it.ID
	go func() {
		saved, err := svc.Update(ctx, it)
		mainthread.Start(func() {
			f.saving = false
			if err != nil {
				if ve := core.AsValidationError(err); ve != nil {
					f.showErrors(ve.Errors)
				} else {
					f.a.dlg.errorBox("Save failed", err.Error())
				}
				f.updateDirty()
				return
			}
			if f.item == nil || f.item.ID != id {
				return
			}
			f.load(saved)
			f.a.pager.Invalidate()
		})
	}()
}

func (f *detailForm) doRevert() {
	if f.item == nil || !f.dirty {
		return
	}
	if !f.a.dlg.confirmDiscard() {
		return
	}
	f.load(*f.item)
}

func (f *detailForm) doCopyRow() {
	if f.item == nil {
		return
	}
	qt.QGuiApplication_Clipboard().SetText(core.ItemCSVRow(*f.item))
	f.a.flash("Row copied")
}

const maxNotesFile = 64 * 1024

func (f *detailForm) doImport() {
	if f.item == nil {
		return
	}
	path := f.a.dlg.openFile()
	if path == "" {
		return
	}
	st, err := os.Stat(path)
	if err != nil {
		f.a.dlg.errorBox("Import Notes", err.Error())
		return
	}
	if st.Size() > maxNotesFile {
		f.a.dlg.errorBox("Import Notes", "File is too large.")
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		f.a.dlg.errorBox("Import Notes", err.Error())
		return
	}
	s := string(b)
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	if utf8.RuneCountInString(s) > 500 {
		s = string([]rune(s)[:500])
	}
	f.notes.SetPlainText(s)
}
