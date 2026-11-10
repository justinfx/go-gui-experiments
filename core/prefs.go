package core

// Theme values for Prefs.Theme.
const (
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

// View values for Prefs.View.
const (
	ViewTable = "table"
	ViewCards = "cards"
)

// Prefs are the user preferences saved between launches.
type Prefs struct {
	Theme        string     `json:"theme"` // "light" | "dark" | "system"
	SortBy       SortField  `json:"sortBy"`
	SortDesc     bool       `json:"sortDesc"`
	Search       string     `json:"search"`
	Categories   []Category `json:"categories"` // empty = all
	Status       *Status    `json:"status"`     // nil = all
	SelectedID   int64      `json:"selectedId"` // 0 = none
	View         string     `json:"view"`       // "table" | "cards"
	WindowWidth  int        `json:"windowWidth"`
	WindowHeight int        `json:"windowHeight"`
}

// PrefsStore loads and saves Prefs.
type PrefsStore interface {
	Load() (Prefs, error) // missing file/key returns DefaultPrefs(), nil
	Save(Prefs) error
}

// DefaultPrefs returns the defaults: system theme, sort by id ascending,
// no filters, nothing selected, table view, 1280 x 800 window.
func DefaultPrefs() Prefs {
	return Prefs{
		Theme:        ThemeSystem,
		SortBy:       SortID,
		View:         ViewTable,
		WindowWidth:  1280,
		WindowHeight: 800,
	}
}

// Query returns the query described by the saved search, filters and sort.
func (p Prefs) Query() Query {
	return Query{Search: p.Search, Categories: p.Categories, Status: p.Status, SortBy: p.SortBy, Desc: p.SortDesc}.Normalize()
}

// SetQuery stores a query's search, filters and sort in the prefs.
func (p *Prefs) SetQuery(q Query) {
	q = q.Normalize()
	p.Search, p.Categories, p.Status, p.SortBy, p.SortDesc = q.Search, q.Categories, q.Status, q.SortBy, q.Desc
}

// Sanitize replaces invalid values with defaults.
func (p Prefs) Sanitize() Prefs {
	d := DefaultPrefs()
	switch p.Theme {
	case ThemeSystem, ThemeLight, ThemeDark:
	default:
		p.Theme = d.Theme
	}
	switch p.View {
	case ViewTable, ViewCards:
	default:
		p.View = d.View
	}
	if !p.SortBy.Valid() {
		p.SortBy = d.SortBy
	}
	if p.WindowWidth < 1024 {
		p.WindowWidth = d.WindowWidth
	}
	if p.WindowHeight < 640 {
		p.WindowHeight = d.WindowHeight
	}
	return p
}
