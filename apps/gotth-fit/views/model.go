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

// Row heights of the virtual list, in px. The table row is the natural height
// of the default shadcn table row (text-sm, p-2, 1 px border); a card is the
// fixed 96 px plus the 8 px gap.
const (
	RowHeightTable = 38
	RowHeightCards = 104
)

// ChunkRows is the number of rows per skippable chunk inside a page block.
const ChunkRows = 50

// MaxVirtualPx caps the height of the scroll space. Browsers stop growing an
// element at roughly 17 to 33 million px, which 1,000,000 table rows exceed,
// so a taller result is scrolled through a proportionally scaled space.
const MaxVirtualPx = 10_000_000

// RowHeight returns the row pitch of a view.
func RowHeight(view string) int {
	if view == "cards" {
		return RowHeightCards
	}
	return RowHeightTable
}

// VirtualHeight is the height of the scroll space for total rows.
func VirtualHeight(total int, view string) int {
	return min(total*RowHeight(view), MaxVirtualPx)
}

// BlockTop is the top of page p in the scroll space when the space is not
// scaled; the client recomputes it from the scroll position (scaled or not).
func BlockTop(view string, total, p int) int64 {
	if total*RowHeight(view) > MaxVirtualPx {
		return 0
	}
	return int64(p) * PageSize * int64(RowHeight(view))
}

// PageVM is everything the full page needs.
type PageVM struct {
	Theme          string // light | dark | system
	View           string
	Desktop        bool
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

// PageURL builds the /page URL for a view, query and page index. The client
// derives the same URL for any page from the data-page-url template.
func PageURL(view string, q core.Query, p int) string {
	return PageURLBase(view, q) + strconv.Itoa(p)
}

// PageURLBase is PageURL up to and including "p=", so the client can append a
// page index without knowing how the query is encoded.
func PageURLBase(view string, q core.Query) string {
	v := q.Values()
	v.Set("view", view)
	return "/page?" + v.Encode() + "&p="
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
