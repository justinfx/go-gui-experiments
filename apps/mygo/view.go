package main

// View types: plain, JSON-friendly shapes that the generated TypeScript
// bindings can express exactly. Every displayed string is produced here with
// the core formatting helpers so the UI never re-implements them.

import (
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"gguix/core"
)

// ItemView is core.Item plus its pre-formatted display values.
type ItemView struct {
	ID          int64   `json:"id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	PriceCents  int64   `json:"priceCents"`
	Quantity    int32   `json:"quantity"`
	Status      string  `json:"status"`
	Rating      float32 `json:"rating"`
	Featured    bool    `json:"featured"`
	RestockDate *string `json:"restockDate"`
	UpdatedAt   string  `json:"updatedAt"`
	Notes       string  `json:"notes"`

	PriceText       string  `json:"priceText"`
	RatingText      string  `json:"ratingText"`
	UpdatedText     string  `json:"updatedText"`
	StockFraction   float64 `json:"stockFraction"`
	StockLow        bool    `json:"stockLow"`
	StockLabel      string  `json:"stockLabel"`
	StarsFull       int     `json:"starsFull"`
	StarsHalf       int     `json:"starsHalf"`
	StarsEmpty      int     `json:"starsEmpty"`
	CategoryInitial string  `json:"categoryInitial"`
	NotesPreview    string  `json:"notesPreview"`
}

// ToView converts a core item.
func ToView(it core.Item) ItemView {
	full, half, empty := core.Stars(it.Rating)
	v := ItemView{
		ID: it.ID, SKU: it.SKU, Name: it.Name, Category: it.Category.String(),
		PriceCents: it.PriceCents, Quantity: it.Quantity, Status: it.Status.String(),
		Rating: it.Rating, Featured: it.Featured,
		UpdatedAt: it.UpdatedAt.UTC().Format(time.RFC3339), Notes: it.Notes,

		PriceText: core.FormatPrice(it.PriceCents), RatingText: core.FormatRating(it.Rating),
		UpdatedText:   core.FormatUpdated(it.UpdatedAt),
		StockFraction: core.StockFraction(it.Quantity), StockLow: core.StockLow(it.Quantity),
		StockLabel: core.StockLabel(it.Quantity),
		StarsFull:  full, StarsHalf: half, StarsEmpty: empty,
		CategoryInitial: core.CategoryInitial(it.Category), NotesPreview: core.NotesPreview(it.Notes),
	}
	if it.RestockDate != nil {
		s := it.RestockDate.String()
		v.RestockDate = &s
	}
	return v
}

// QueryArg is core.Query with string enums.
type QueryArg struct {
	Search     string   `json:"search"`
	Categories []string `json:"categories"`
	Status     string   `json:"status"` // "" = all
	SortBy     string   `json:"sortBy"`
	Desc       bool     `json:"desc"`
}

// ToQuery converts to a core query.
func (a QueryArg) ToQuery() (core.Query, error) {
	q := core.Query{Search: a.Search, Desc: a.Desc}
	for _, n := range a.Categories {
		c, err := core.ParseCategory(n)
		if err != nil {
			return core.Query{}, err
		}
		q.Categories = append(q.Categories, c)
	}
	if a.Status != "" {
		s, err := core.ParseStatus(a.Status)
		if err != nil {
			return core.Query{}, err
		}
		q.Status = &s
	}
	f, err := core.ParseSortField(a.SortBy)
	if err != nil {
		return core.Query{}, err
	}
	q.SortBy = f
	return q.Normalize(), nil
}

// PageArg is core.PageRequest.
type PageArg struct {
	Query  QueryArg `json:"query"`
	Offset int      `json:"offset"`
	Limit  int      `json:"limit"`
}

// PageView is core.Page plus the formatted total.
type PageView struct {
	Offset    int        `json:"offset"`
	Items     []ItemView `json:"items"`
	Total     int        `json:"total"`
	TotalText string     `json:"totalText"`
	QueryKey  string     `json:"queryKey"`
}

// FieldErr is core.FieldError.
type FieldErr struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ItemEdit is the form content, as typed.
type ItemEdit struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Price       string  `json:"price"` // as typed, parsed with core.ParsePrice
	Quantity    int64   `json:"quantity"`
	Status      string  `json:"status"`
	Rating      float64 `json:"rating"`
	Featured    bool    `json:"featured"`
	RestockDate string  `json:"restockDate"` // "" = none
	Notes       string  `json:"notes"`
}

// UpdateResult is either the stored item or the field errors.
type UpdateResult struct {
	Item   *ItemView  `json:"item"`
	Errors []FieldErr `json:"errors"`
}

// buildItem parses an edit into a core.Item. Parse failures are returned as
// field errors; the offending field gets a neutral value so that the other
// fields can still be validated.
func buildItem(e ItemEdit) (core.Item, []FieldErr) {
	var errs []FieldErr
	it := core.Item{ID: e.ID, SKU: core.SKUFor(e.ID), Name: e.Name, Featured: e.Featured, Notes: e.Notes}
	if c, err := core.ParseCategory(e.Category); err == nil {
		it.Category = c
	} else {
		errs = append(errs, FieldErr{"category", err.Error()})
	}
	if s, err := core.ParseStatus(e.Status); err == nil {
		it.Status = s
	} else {
		errs = append(errs, FieldErr{"status", err.Error()})
	}
	if cents, err := core.ParsePrice(e.Price); err == nil {
		it.PriceCents = cents
	} else {
		errs = append(errs, FieldErr{"priceCents", err.Error()})
	}
	switch {
	case e.Quantity > math.MaxInt32:
		it.Quantity = math.MaxInt32
	case e.Quantity < math.MinInt32:
		it.Quantity = math.MinInt32
	default:
		it.Quantity = int32(e.Quantity)
	}
	it.Rating = float32(math.Round(e.Rating*10) / 10)
	if strings.TrimSpace(e.RestockDate) != "" {
		if d, err := core.ParseDate(e.RestockDate); err == nil {
			it.RestockDate = &d
		} else {
			errs = append(errs, FieldErr{"restockDate", err.Error()})
		}
	}
	return it, errs
}

// validateEdit returns the combined parse and validation errors.
func validateEdit(e ItemEdit) (core.Item, []FieldErr) {
	it, errs := buildItem(e)
	if ve := core.ValidateItem(it); ve != nil {
		for _, fe := range ve.Errors {
			// A price that did not parse already has its own message.
			if fe.Field == "priceCents" && slices.ContainsFunc(errs, func(x FieldErr) bool { return x.Field == "priceCents" }) {
				continue
			}
			errs = append(errs, FieldErr{fe.Field, fe.Message})
		}
	}
	return it, errs
}

// PrefsView is core.Prefs with string enums.
type PrefsView struct {
	Theme        string   `json:"theme"`
	SortBy       string   `json:"sortBy"`
	SortDesc     bool     `json:"sortDesc"`
	Search       string   `json:"search"`
	Categories   []string `json:"categories"`
	Status       string   `json:"status"`
	SelectedID   int64    `json:"selectedId"`
	View         string   `json:"view"`
	WindowWidth  int      `json:"windowWidth"`
	WindowHeight int      `json:"windowHeight"`
}

func prefsToView(p core.Prefs) PrefsView {
	v := PrefsView{
		Theme: p.Theme, SortBy: string(p.SortBy), SortDesc: p.SortDesc, Search: p.Search,
		Categories: []string{}, SelectedID: p.SelectedID, View: p.View,
		WindowWidth: p.WindowWidth, WindowHeight: p.WindowHeight,
	}
	for _, c := range p.Categories {
		v.Categories = append(v.Categories, c.String())
	}
	if p.Status != nil {
		v.Status = p.Status.String()
	}
	return v
}

func prefsFromView(v PrefsView) (core.Prefs, error) {
	q, err := QueryArg{Search: v.Search, Categories: v.Categories, Status: v.Status, SortBy: v.SortBy, Desc: v.SortDesc}.ToQuery()
	if err != nil {
		return core.Prefs{}, err
	}
	p := core.Prefs{
		Theme: v.Theme, SortBy: q.SortBy, SortDesc: q.Desc, Search: v.Search, Categories: q.Categories,
		Status: q.Status, SelectedID: v.SelectedID, View: v.View,
		WindowWidth: v.WindowWidth, WindowHeight: v.WindowHeight,
	}
	return p.Sanitize(), nil
}

// HostView is host.Info for the host panel.
type HostView struct {
	Source    string    `json:"source"`
	Username  string    `json:"username"`
	FullName  string    `json:"fullName"`
	UID       string    `json:"uid"`
	HomeDir   string    `json:"homeDir"`
	Hostname  string    `json:"hostname"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	PID       int       `json:"pid"`
	ConfigDir string    `json:"configDir"`
	Locale    string    `json:"locale"`
	Notes     []string  `json:"notes"`
	Identity  string    `json:"identity"` // "<Username>@<Hostname>" or "unknown user"
	Rows      []HostRow `json:"rows"`     // panel lines in display order
	Actions   bool      `json:"actions"`  // reveal actions available
}

// HostRow is one label/value line.
type HostRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

var errUnsupported = errors.New("Not available in the browser")

// BadgePair is a background and text color.
type BadgePair struct {
	Background string `json:"background"`
	Text       string `json:"text"`
}

// PaletteView is core.Palette with plain fields.
type PaletteView struct {
	Active       BadgePair `json:"active"`
	Backordered  BadgePair `json:"backordered"`
	Discontinued BadgePair `json:"discontinued"`
	StockTrack   string    `json:"stockTrack"`
	StockFill    string    `json:"stockFill"`
	StockFillLow string    `json:"stockFillLow"`
	StarFilled   string    `json:"starFilled"`
	StarEmpty    string    `json:"starEmpty"`
	Featured     BadgePair `json:"featured"`
	Tiles        []string  `json:"tiles"`
	TileText     string    `json:"tileText"`
}

// PaletteSet holds both themes' custom-rendering colors (BRIEF 4.5.1).
type PaletteSet struct {
	Light PaletteView `json:"light"`
	Dark  PaletteView `json:"dark"`
}

func paletteView(p core.Palette) PaletteView {
	b := func(c core.BadgeColors) BadgePair { return BadgePair{c.Background, c.Text} }
	return PaletteView{
		Active: b(p.Active), Backordered: b(p.Backordered), Discontinued: b(p.Discontinued),
		StockTrack: p.StockTrack, StockFill: p.StockFill, StockFillLow: p.StockFillLow,
		StarFilled: p.StarFilled, StarEmpty: p.StarEmpty, Featured: b(p.Featured),
		Tiles: p.CategoryTiles[:], TileText: p.TileText,
	}
}
