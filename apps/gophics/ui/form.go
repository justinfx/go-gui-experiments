package ui

import (
	"strconv"
	"time"

	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/layout"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
)

func categoryNames() []string {
	out := make([]string, len(core.AllCategories))
	for i, c := range core.AllCategories {
		out[i] = c.String()
	}
	return out
}

func (s *App) formView(th theme.Theme) widget.Widget {
	on := s.HasItem
	f := s.Form
	edit := func(fn func(f *Form)) {
		if on {
			s.SetState(func() { s.EditForm(fn) })
		}
	}
	errText := func(field string) widget.Widget {
		if msg, ok := s.Errors[field]; ok {
			return widget.Padding{Insets: geom.Insets{Top: 2}, Child: widget.Text{Value: msg, Size: th.Type.Caption, Color: th.Danger, Wrap: true}}
		}
		return nil
	}
	field := func(label string, control widget.Widget, errField string) widget.Widget {
		ctrl := widget.Column(control)
		if e := errText(errField); e != nil {
			ctrl = widget.Column(control, e)
		}
		ctrl.CrossAlign = layout.CrossStretch
		r := widget.Row(
			widget.Sized{W: 100, Child: widget.Padding{Insets: geom.Insets{Top: 8},
				Child: widget.Text{Value: label, Size: th.Type.Label, Color: th.Muted}}},
			widget.Expand(ctrl),
		)
		r.CrossAlign = layout.CrossStart
		return widget.Padding{Insets: geom.Insets{Bottom: 8}, Child: r}
	}
	ro := func(v string) widget.Widget {
		return widget.Padding{Insets: geom.Insets{Top: 8}, Child: widget.Text{Value: v, Size: th.Type.Label, Color: th.Text}}
	}
	txt := func(v string, set func(f *Form, v string)) widget.Widget {
		return Field{OnKeyPreview: s.fieldKeyPreview, Value: v, OnChange: func(nv string) { edit(func(f *Form) { set(f, nv) }) }}
	}

	id, sku, updated := "", "", ""
	if on {
		id, sku, updated = strconv.FormatInt(s.Item.ID, 10), s.Item.SKU, core.FormatUpdated(s.Item.UpdatedAt)
	}
	radios := []widget.Widget{}
	for i, st := range core.AllStatuses {
		i := i
		radios = append(radios, theme.Radio{Selected: on && f.Status == i, Label: st.String(),
			OnSelect: func() { edit(func(f *Form) { f.Status = i }) }}, gapW(12))
	}
	catSel := -1
	if on {
		catSel = f.Category
	}
	rating := widget.Row(
		widget.Expand(theme.Slider{Value: float32(f.Rating) / 50, Label: "Rating",
			OnChange: func(v float32) { edit(func(f *Form) { f.Rating = int(v*50 + 0.5) }) }}),
		gapW(8),
		widget.Sized{W: 32, Child: widget.Text{Value: core.FormatRating(core.RatingFromTenths(f.Rating)), Size: th.Type.Label, Color: th.Text}},
	)
	restock := row(
		widget.Expand(Field{OnKeyPreview: s.fieldKeyPreview, Value: f.Restock, Placeholder: "YYYY-MM-DD",
			OnChange: func(v string) { edit(func(f *Form) { f.Restock = v }) }}),
		gapW(6),
		theme.Button{Label: "Pick…", OnTap: func() {
			if on {
				s.SetState(s.pickRestockDate)
			}
		}},
		gapW(6),
		theme.Button{Label: "Clear", OnTap: func() { edit(func(f *Form) { f.Restock = "" }) }},
	)
	notes := widget.Sized{H: 4*20 + 16, Child: Field{OnKeyPreview: s.fieldKeyPreview, Value: f.Notes, Multiline: true,
		OnChange: func(v string) { edit(func(f *Form) { f.Notes = v }) }}}

	dirty := s.Dirty()
	var saveTap, revertTap func()
	if dirty {
		saveTap, revertTap = s.act(s.Save), s.act(s.Revert)
	}
	buttons := []widget.Widget{
		disabled(!dirty, theme.Button{Label: "Save", Primary: true, OnTap: saveTap}), gapW(8),
		disabled(!dirty, theme.Button{Label: "Revert", OnTap: revertTap}), gapW(8),
		theme.Button{Label: "Copy Row", OnTap: s.act(s.CopyRow)}, gapW(8),
		theme.Button{Label: "Import Notes…", OnTap: s.act(s.ImportNotes)},
	}
	if dirty {
		buttons = append(buttons, gapW(12), widget.Text{Value: "Unsaved changes", Size: th.Type.Label, Color: th.Muted})
	}

	form := widget.Column(
		field("ID", ro(id), ""),
		field("SKU", ro(sku), ""),
		field("Name", txt(f.Name, func(f *Form, v string) { f.Name = v }), "name"),
		field("Category", Select{Options: categoryNames(), Selected: catSel,
			OnChange: func(i int) { edit(func(f *Form) { f.Category = i }) }}, "category"),
		field("Price", txt(f.Price, func(f *Form, v string) { f.Price = v }), "priceCents"),
		field("Quantity", txt(f.Qty, func(f *Form, v string) { f.Qty = v }), "quantity"),
		field("Status", widget.Padding{Insets: geom.Insets{Top: 6}, Child: widget.Row(radios...)}, "status"),
		field("Rating", rating, "rating"),
		field("Featured", widget.Padding{Insets: geom.Insets{Top: 4}, Child: theme.Checkbox{Checked: f.Featured, Label: "",
			OnChange: func(v bool) { edit(func(f *Form) { f.Featured = v }) }}}, ""),
		field("Restock date", restock, "restockDate"),
		field("Updated", ro(updated), ""),
		field("Notes", notes, "notes"),
	)
	form.CrossAlign = layout.CrossStretch

	head := []widget.Widget{}
	if !on {
		head = append(head, widget.Padding{Insets: geom.Insets{Bottom: 12},
			Child: widget.Text{Value: "Select an item to edit.", Size: th.Type.Body, Color: th.Text}})
	}
	all := widget.Column(append(head,
		disabled(!on, form),
		gapH(4),
		disabled(!on, row(buttons...)),
	)...)
	all.CrossAlign = layout.CrossStretch
	return widget.Scroll{Child: widget.Padding{All: 16, Child: all}}
}

func (s *App) pickRestockDate() {
	initial := time.Now()
	if d, err := core.ParseDate(s.Form.Restock); err == nil {
		initial = time.Date(d.Year, d.Month, d.Day, 12, 0, 0, 0, time.UTC)
	}
	theme.ShowDatePicker(s.ctx, initial, func(t time.Time) {
		s.SetState(func() { s.EditForm(func(f *Form) { f.Restock = core.DateOf(t).String() }) })
	})
}
