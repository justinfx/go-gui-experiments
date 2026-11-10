// Package views holds the templ components and view models of the GoTTH app.
package views

import (
	"net/url"
	"strconv"

	"gguix/core"
	"gguix/core/host"
)

// PageSize is the number of rows per server-rendered page.
const PageSize = 200

// MaxLoadedPages is the number of pages kept in the DOM.
const MaxLoadedPages = 5

// PageVM is everything the full page needs.
type PageVM struct {
	Theme          string // light | dark | system
	View           string
	Desktop        bool
	NativeMenu     bool // the desktop shell has a native menu bar
	Bench          bool
	ProcessStartMs int64
	Query          core.Query
	Filters        FilterVM
	List           ListVM
	Form           FormVM
	Host           host.Info
	GoVersion      string
	BuildMarker    string
	TemplVersion   string
	UIVersion      string
}

// FilterVM is the toolbar state.
type FilterVM struct {
	Search     string
	Categories []core.Category
	Status     string // "" or status name
	Total      int
	View       string
	Theme      string
}

// HasCategory reports whether c is selected.
func (f FilterVM) HasCategory(c core.Category) bool {
	for _, x := range f.Categories {
		if x == c {
			return true
		}
	}
	return false
}

// CategoryLabel is the label of the Categories button.
func (f FilterVM) CategoryLabel() string {
	if len(f.Categories) == 0 {
		return "Categories"
	}
	return "Categories (" + strconv.Itoa(len(f.Categories)) + ")"
}

// ListVM describes one list window: the first page and how to continue.
type ListVM struct {
	View       string
	Query      core.Query
	Page       core.Page // the first page of the window
	PageIdx    int       // index of Page
	LastPage   int       // index of the last page of the result (-1 when empty)
	SelectedID int64
	ScrollRow  int // absolute row index that becomes the top row, -1 for none
	FetchMs    float64
	Sort       core.SortField
	Desc       bool
}

// PageURL returns the URL that loads page p in the given direction.
func (l ListVM) PageURL(p int, dir string) string {
	return PageURL(l.View, l.Query, p, dir)
}

// PageURL builds the /page URL for a view, query, page index and direction
// (down, up or fill).
func PageURL(view string, q core.Query, p int, dir string) string {
	v := q.Values()
	v.Set("p", strconv.Itoa(p))
	v.Set("dir", dir)
	v.Set("view", view)
	return "/page?" + v.Encode()
}

// QueryString encodes a query for a URL.
func QueryString(q core.Query) string { return url.Values(q.Values()).Encode() }

// FormVM is the state of the detail form.
type FormVM struct {
	Item       *core.Item // nil when nothing is selected
	Values     FormValues
	Errors     map[string]string
	ForceDirty bool
	CSV        string
}

// FormValues are the string values shown in the form controls.
type FormValues struct {
	Name, Category, Price, Quantity, Status, Rating, Restock, Notes string
	Featured                                                        bool
}

// ValuesFromItem fills form values from an item.
func ValuesFromItem(it core.Item) FormValues {
	v := FormValues{
		Name:     it.Name,
		Category: it.Category.String(),
		Price:    core.FormatPrice(it.PriceCents),
		Quantity: strconv.Itoa(int(it.Quantity)),
		Status:   it.Status.String(),
		Rating:   core.FormatRating(it.Rating),
		Featured: it.Featured,
		Notes:    it.Notes,
	}
	if it.RestockDate != nil {
		v.Restock = it.RestockDate.String()
	}
	return v
}

// ExportVM is the export status shown in the status bar.
type ExportVM struct {
	State string // idle | running | done | cancelled | error
	ID    int
	Done  int
	Total int
	Rows  int
	Err   string
}

// Percent returns the progress percentage.
func (e ExportVM) Percent() int {
	if e.Total <= 0 {
		return 0
	}
	return min(100, e.Done*100/e.Total)
}
