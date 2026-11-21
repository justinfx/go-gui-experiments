// Package ui holds the GGUIX reference app built with Gophics widgets.
//
// The root State (App) is the presentation layer: every user action is a
// method on it (SetSearch, SortBy, SelectRow, Save, ...), the widget tree in
// Build only renders its fields, and tests drive the same methods through a
// headless app (app.NewHeadless).
package ui

import (
	"context"
	"strconv"
	"time"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/doug/gophics/app"
	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/layout"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"

	"gguix/core"
	"gguix/core/host"
)

// AppID is the core prefs/host app identifier.
const AppID = "gophics"

// Title is the window title.
const Title = "GGUIX - Gophics"

// Table and card extents. The table row height is pinned to 32 px (the theme
// default would be Body+18 = 33 px) because LazyList sums row heights in
// float32: a power-of-two extent keeps every offset exact up to 1,000,000 rows.
const (
	TableRowH  = 32
	CardH      = 96
	CardGap    = 8
	CardExtent = CardH + CardGap
	headerH    = 36
)

// Options configure the root widget.
type Options struct {
	Svc   core.DataService
	Store core.PrefsStore
	Host  host.Info
	Bench bool
	// Prefs are the prefs loaded at startup.
	Prefs core.Prefs
	// Ready is called once on the UI thread after the first build, with the
	// app state. The bench driver and tests use it.
	Ready func(*App)
	// App is set to the root state once it exists (main uses it to save
	// prefs on quit).
	App *App
}

// Root is the app's root widget.
type Root struct{ Opts *Options }

func (r Root) CreateState() widget.State {
	a := &App{opts: r.Opts}
	r.Opts.App = a
	return a
}

// Config builds the app.Config for a window of the given logical size.
func Config(w, h int) app.Config {
	cfg := app.Config{
		Title:          Title,
		AppID:          "com.gguix.gophics",
		Size:           geom.Size{W: float32(w), H: float32(h)},
		Background:     theme.Light().Bg,
		BackgroundDark: theme.Dark().Bg,
		Font:           goregular.TTF,
		FontFamilies:   map[string][]byte{theme.FontBold: gobold.TTF, "mono": gomono.TTF},
	}
	platformConfig(&cfg)
	return cfg
}

// Form holds the editable draft of the selected item as control values.
type Form struct {
	Name     string
	Category int
	Price    string
	Qty      string
	Status   int
	Rating   int // tenths, 0..50
	Featured bool
	Restock  string // YYYY-MM-DD or ""
	Notes    string
}

func formFrom(it core.Item) Form {
	f := Form{
		Name:     it.Name,
		Category: int(it.Category),
		Price:    core.FormatPrice(it.PriceCents),
		Qty:      strconv.Itoa(int(it.Quantity)),
		Status:   int(it.Status),
		Rating:   core.RatingTenths(it.Rating),
		Featured: it.Featured,
		Notes:    it.Notes,
	}
	if it.RestockDate != nil {
		f.Restock = it.RestockDate.String()
	}
	return f
}

// App is the root State and the presentation layer.
type App struct {
	widget.StateBase[Root]
	opts *Options

	ctx     widget.Ctx
	post    func(func())
	started bool

	svc   core.DataService
	store core.PrefsStore
	Host  host.Info
	Pager *core.Pager

	Prefs  core.Prefs
	Query  core.Query
	Search string // the search field's live text (Query.Search lags by the debounce)
	View   string
	Theme  string

	// Selection and form.
	SelectedID int64
	Item       core.Item // the loaded (saved) item
	HasItem    bool
	Form       Form
	Errors     map[string]string
	saving     bool

	// Scroll state.
	tableCtl  *widget.ScrollController
	cardsCtl  *widget.ScrollController
	viewportH float32
	First     int
	Last      int

	pendingSelectRow int  // row to select once loaded (go-to-row, restore); -1 none
	pendingTopRow    int  // row to scroll to the top after the next layout; -1 none
	checkSelReload   bool // after a query change: keep selection only if on first page
	GoToText         string

	// Export.
	Exporting    bool
	ExportDone   int
	ExportTotal  int
	exportCancel context.CancelFunc
	LastExport   string

	// Transient status text.
	Message  string
	msgTimer *time.Timer

	// Dialog bookkeeping (tests read DialogTitle).
	DialogTitle   string
	ExportName    string // desktop export file name prompt
	dialogDismiss func()

	split      float32
	winW, winH int

	searchTimer *time.Timer
	prefsTimer  *time.Timer

	menusPublished string
	pagerDirty     bool
	// UpdateCalls counts DataService.Update calls (tests).
	UpdateCalls int

	searchGen      int
	listTop        float32
	hOff           float32 // table horizontal scroll offset
	hCtl           *widget.ScrollController
	hdrCtl         *widget.ScrollController
	catsAnchor     geom.Pt
	colW           []float32
	pendingDiscard func()
	MessageBody    string
	LastNotice     string
}

// FocusSearch remounts the search field with Autofocus (Gophics has no
// imperative focus API; Autofocus fires once per mount).
func (s *App) FocusSearch() { s.searchGen++ }

func (s *App) Init(ctx widget.Ctx) {
	o := s.opts
	s.svc, s.store, s.Host = o.Svc, o.Store, o.Host
	s.tableCtl = &widget.ScrollController{}
	s.hCtl = &widget.ScrollController{}
	s.hdrCtl = &widget.ScrollController{}
	s.cardsCtl = &widget.ScrollController{}
	s.split = 0.65
	s.pendingSelectRow, s.pendingTopRow = -1, -1
	s.First, s.Last = 0, -1
	s.Errors = map[string]string{}
	s.post = ctx.Post()

	// Prefs are loaded by main before the window exists: on js/wasm a remote
	// Load is a blocking fetch, which must not run on the event goroutine.
	p := o.Prefs
	s.Prefs = p
	s.Theme = p.Theme
	if s.Theme == "" {
		s.Theme = core.ThemeSystem
	}
	s.View = p.View
	if s.View != core.ViewCards {
		s.View = core.ViewTable
	}
	s.winW, s.winH = p.WindowWidth, p.WindowHeight
	s.Search = p.Search
	s.Query = core.Query{Search: p.Search, Categories: p.Categories, Status: p.Status, SortBy: p.SortBy, Desc: p.SortDesc}
	if o.Bench {
		s.View = core.ViewTable
	}

	s.Pager = core.NewPager(s.svc, core.DefaultPagerOptions(), s.onPagerChange)
	s.Pager.SetQuery(s.Query)

	if p.SelectedID != 0 && !o.Bench {
		s.restoreSelection(p.SelectedID)
	}
}

func (s *App) Dispose() {
	s.SavePrefsNow()
	if s.Pager != nil {
		s.Pager.Close()
	}
}

// onPagerChange runs on a pager goroutine; marshal onto the UI thread,
// coalescing bursts into one rebuild.
func (s *App) onPagerChange() {
	s.post(func() {
		if s.pagerDirty {
			return
		}
		s.pagerDirty = true
		s.SetState(func() {
			s.pagerDirty = false
			s.afterPagerChange()
		})
	})
}

func (s *App) afterPagerChange() {
	total := s.Pager.Total()
	if s.checkSelReload && total >= 0 {
		end := min(total, core.DefaultPagerOptions().PageSize) - 1
		if end < 0 || s.Pager.RangeLoaded(0, end) {
			s.checkSelReload = false
			found := false
			for i := 0; i <= end; i++ {
				if it, ok := s.Pager.Row(i); ok && it.ID == s.SelectedID {
					found = true
					break
				}
			}
			if !found {
				s.clearSelection()
			}
		}
	}
	if r := s.pendingSelectRow; r >= 0 && total >= 0 {
		if it, ok := s.Pager.Row(r); ok {
			s.pendingSelectRow = -1
			s.selectItem(it)
		} else if r >= total {
			s.pendingSelectRow = -1
		}
	}
}

// Build renders the whole window.
func (s *App) Build(ctx widget.Ctx) widget.Widget {
	s.ctx = ctx
	th := s.themeFor(ctx)
	if !s.started {
		s.started = true
		ready := s.opts.Ready
		s.post(func() {
			if ready != nil {
				ready(s)
			}
		})
	}
	s.publishMenus()
	if r := s.pendingTopRow; r >= 0 {
		// Scroll after this frame's layout so MaxOffset covers the new Total.
		s.pendingTopRow = -1
		s.post(func() { s.scrollRowToTop(r) })
	}

	body := widget.Column(
		s.toolbar(th),
		widget.Fill{Color: th.Border, Child: widget.Sized{H: 1}},
		widget.Expand(s.mainArea(th)),
		widget.Fill{Color: th.Border, Child: widget.Sized{H: 1}},
		s.statusBar(th),
	)
	body.CrossAlign = layout.CrossStretch

	root := widget.LayoutBuilder{Build: func(cs layout.Constraints) widget.Widget {
		s.noteWindowSize(cs.Max)
		return widget.Interactive{
			Gestures: widget.Gestures{OnKey: s.onKey},
			Child:    body,
		}
	}}
	return widget.Provide[theme.Theme]{Value: th, Child: widget.Fill{Color: th.Bg, Child: root}}
}

func (s *App) themeFor(ctx widget.Ctx) theme.Theme {
	dark := false
	switch s.Theme {
	case core.ThemeDark:
		dark = true
	case core.ThemeLight:
	default:
		dark = ctx.DarkMode()
	}
	if dark {
		return theme.Dark()
	}
	return theme.Light()
}

// IsDark reports the effective theme.
func (s *App) IsDark() bool { return s.themeFor(s.ctx).Dark }

func (s *App) noteWindowSize(sz geom.Size) {
	if s.opts.Bench || sz.W <= 0 || sz.H <= 0 || sz.W > 1e6 || sz.H > 1e6 {
		return
	}
	w, h := int(sz.W+0.5), int(sz.H+0.5)
	if w != s.winW || h != s.winH {
		s.winW, s.winH = w, h
		s.post(s.schedulePrefsSave)
	}
}

// ---- prefs ----

func (s *App) currentPrefs() core.Prefs {
	p := s.Prefs
	p.Theme = s.Theme
	p.View = s.View
	p.SortBy = s.Query.SortBy
	if p.SortBy == "" {
		p.SortBy = core.SortID
	}
	p.SortDesc = s.Query.Desc
	p.Search = s.Query.Search
	p.Categories = s.Query.Categories
	p.Status = s.Query.Status
	p.SelectedID = s.SelectedID
	if s.winW > 0 && s.winH > 0 {
		p.WindowWidth, p.WindowHeight = s.winW, s.winH
	}
	return p
}

func (s *App) schedulePrefsSave() {
	if s.prefsTimer != nil {
		s.prefsTimer.Stop()
	}
	s.prefsTimer = time.AfterFunc(500*time.Millisecond, func() { s.post(s.SavePrefsNow) })
}

// SavePrefsNow writes the prefs through the core PrefsStore.
func (s *App) SavePrefsNow() {
	if s.prefsTimer != nil {
		s.prefsTimer.Stop()
	}
	p := s.currentPrefs()
	s.Prefs = p
	// Off the UI goroutine: the remote store is an HTTP PUT, which would
	// deadlock the js/wasm event loop if it blocked here.
	go func() { _ = s.store.Save(p) }()
}

// SavePrefsSync writes prefs synchronously (on quit, from main).
func (s *App) SavePrefsSync() { _ = s.store.Save(s.currentPrefs()) }

// Flash shows text in the status bar for 2 s.
func (s *App) Flash(msg string) {
	s.Message = msg
	if s.msgTimer != nil {
		s.msgTimer.Stop()
	}
	s.msgTimer = time.AfterFunc(2*time.Second, func() {
		s.post(func() { s.SetState(func() { s.Message = "" }) })
	})
}

// Post runs fn on the UI thread with a rebuild (any goroutine).
func (s *App) Post(fn func()) { s.post(func() { s.SetState(fn) }) }
