package gotth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"

	"gguix/apps/gotth/assets"
	"gguix/apps/gotth/static"
	"gguix/apps/gotth/views"
	"gguix/core"
	"gguix/core/host"
)

// AppID is the application id used for prefs and host info.
const AppID = "gotth"

// FrameworkVersion is shown in the About dialog.
const FrameworkVersion = "GoTTH: templ v0.3.1100, shadcn-templ v2.0.0, htmx 2.0.11, Tailwind v4.3.5"

// Options configures a Server.
type Options struct {
	Svc   core.DataService
	Prefs core.PrefsStore
	// Native is non-nil in the desktop shell (desktop mode).
	Native Native
	// Bench turns bench mode on for every page (desktop). The web target
	// turns it on per page with ?bench=1.
	Bench       bool
	BenchExport string // desktop bench: file the export is written to
	Quit        func() // desktop: closes the window
	// NativeMenu is true when the desktop shell installed a native menu bar,
	// so the page does not draw its own.
	NativeMenu bool
	SaveDelay   time.Duration
}

// Server is the GoTTH HTTP server state: one user, one set of prefs.
type Server struct {
	opts Options
	mux  *http.ServeMux
	host host.Info

	mu       sync.Mutex
	prefs    core.Prefs
	saveTime *time.Timer
	unsaved  bool

	exMu       sync.Mutex
	exSeq      int
	ex         views.ExportVM
	exCancel   context.CancelFunc
	exSubs     map[chan struct{}]struct{}
	lastExport string
}

// New creates the server and loads the prefs.
func New(opts Options) (*Server, error) {
	if opts.SaveDelay == 0 {
		opts.SaveDelay = 500 * time.Millisecond
	}
	p, err := opts.Prefs.Load()
	if err != nil {
		p = core.DefaultPrefs()
	}
	s := &Server{opts: opts, prefs: p.Sanitize(), exSubs: map[chan struct{}]struct{}{}, ex: views.ExportVM{State: "idle"}}
	if opts.Native != nil {
		s.host = host.Current(AppID)
	} else {
		s.host = host.Server(AppID)
	}
	s.mux = http.NewServeMux()
	s.routes()
	if opts.Bench {
		s.kickBenchWatchdog()
	}
	return s, nil
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	if os.Getenv("GOTTH_LOG") == "1" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t0 := time.Now()
			fmt.Fprintf(os.Stderr, "REQ  %s %s\n", r.Method, r.URL.RequestURI())
			s.mux.ServeHTTP(w, r)
			fmt.Fprintf(os.Stderr, "DONE %s %s %s\n", r.Method, r.URL.Path, time.Since(t0).Round(time.Millisecond))
		})
	}
	return s.mux
}

// Prefs returns a copy of the current prefs.
func (s *Server) Prefs() core.Prefs {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prefs
}

// Desktop reports whether the server runs inside the desktop shell.
func (s *Server) Desktop() bool { return s.opts.Native != nil }

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /{$}", s.handleIndex)
	m.HandleFunc("POST /query", s.handleQuery)
	m.HandleFunc("POST /view", s.handleView)
	m.HandleFunc("POST /goto", s.handleGoto)
	m.HandleFunc("GET /page", s.handlePage)
	m.HandleFunc("POST /select", s.handleSelect)
	m.HandleFunc("GET /form", s.handleForm)
	m.HandleFunc("POST /save", s.handleSave)
	m.HandleFunc("POST /theme", s.handleTheme)
	m.HandleFunc("POST /prefs/window", s.handleWindow)
	m.HandleFunc("GET /host", s.handleHost)
	m.HandleFunc("GET /palette.css", s.handlePalette)
	m.HandleFunc("GET /export", s.handleExportWeb)
	m.HandleFunc("GET /export/progress", s.handleExportProgress)
	m.HandleFunc("POST /export/cancel", s.handleExportCancel)
	m.HandleFunc("POST /bench/metric", s.handleBenchMetric)
	m.HandleFunc("POST /bench/quit", s.handleBenchQuit)
	// gguix:target desktop
	if s.Desktop() {
		m.HandleFunc("POST /export/save", s.handleExportSave)
		m.HandleFunc("POST /native/open-notes", s.handleNativeOpenNotes)
		m.HandleFunc("POST /native/copy", s.handleNativeCopy)
		m.HandleFunc("POST /native/quit", s.handleNativeQuit)
		m.HandleFunc("POST /host/reveal-config", s.handleRevealConfig)
		m.HandleFunc("POST /host/reveal-export", s.handleRevealExport)
	}
	// gguix:end
	m.Handle("GET /assets/", http.StripPrefix("/assets/", noCache(s.fileServer(assets.Assets, "assets"))))
	m.Handle("GET /static/", http.StripPrefix("/static/", noCache(s.fileServer(static.FS, "static"))))
}

// fileServer serves the embedded files, or the files on disk when
// GOTTH_DEV=1 (make dev), so edits show without a rebuild.
func (s *Server) fileServer(efs fs.FS, dir string) http.Handler {
	if os.Getenv("GOTTH_DEV") == "1" {
		return http.FileServer(http.Dir(dir))
	}
	return http.FileServer(http.FS(efs))
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------- prefs

// update changes the prefs and schedules a save 500 ms after the last change.
func (s *Server) update(f func(p *core.Prefs)) core.Prefs {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(&s.prefs)
	s.unsaved = true
	if s.saveTime != nil {
		s.saveTime.Stop()
	}
	s.saveTime = time.AfterFunc(s.opts.SaveDelay, func() { _ = s.Flush() })
	return s.prefs
}

// Flush saves the prefs now if they changed.
func (s *Server) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveTime != nil {
		s.saveTime.Stop()
		s.saveTime = nil
	}
	if !s.unsaved {
		return nil
	}
	s.unsaved = false
	return s.opts.Prefs.Save(s.prefs)
}

// ---------------------------------------------------------------- helpers

func render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	var buf bytes.Buffer
	if err := c.Render(r.Context(), &buf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func renderString(ctx context.Context, c templ.Component) string {
	var buf bytes.Buffer
	_ = c.Render(ctx, &buf)
	return buf.String()
}

func (s *Server) filterVM(p core.Prefs, total int) views.FilterVM {
	f := views.FilterVM{Search: p.Search, Categories: p.Query().Categories, Total: total, View: p.View, Theme: p.Theme}
	if p.Status != nil {
		f.Status = p.Status.String()
	}
	return f
}

// listVM loads the page that starts the window and describes it.
func (s *Server) listVM(ctx context.Context, p core.Prefs, startPage, scrollRow int) (views.ListVM, error) {
	q := p.Query()
	if startPage < 0 {
		startPage = 0
	}
	t0 := time.Now()
	page, err := s.opts.Svc.Page(ctx, core.PageRequest{Query: q, Offset: startPage * views.PageSize, Limit: views.PageSize})
	if err != nil {
		return views.ListVM{}, err
	}
	if len(page.Items) == 0 && page.Total > 0 && startPage > 0 {
		startPage = (page.Total - 1) / views.PageSize
		t0 = time.Now()
		page, err = s.opts.Svc.Page(ctx, core.PageRequest{Query: q, Offset: startPage * views.PageSize, Limit: views.PageSize})
		if err != nil {
			return views.ListVM{}, err
		}
	}
	return views.ListVM{
		View:       p.View,
		Query:      q,
		Page:       page,
		PageIdx:    startPage,
		LastPage:   lastPage(page.Total),
		SelectedID: p.SelectedID,
		ScrollRow:  scrollRow,
		FetchMs:    core.Ms(time.Since(t0)),
		Sort:       q.SortBy,
		Desc:       q.Desc,
	}, nil
}

func lastPage(total int) int {
	if total <= 0 {
		return -1
	}
	return (total - 1) / views.PageSize
}

func (s *Server) formFor(ctx context.Context, id int64) (views.FormVM, error) {
	if id == 0 {
		return views.FormVM{}, nil
	}
	it, err := s.opts.Svc.Get(ctx, id)
	if err != nil {
		return views.FormVM{}, err
	}
	return views.FormVM{Item: &it, Values: views.ValuesFromItem(it), CSV: core.ItemCSVRow(it)}, nil
}

func (s *Server) theme(r *http.Request, p core.Prefs) string {
	if c, err := r.Cookie("gguix-theme"); err == nil {
		switch c.Value {
		case core.ThemeDark, core.ThemeLight, core.ThemeSystem:
			return c.Value
		}
	}
	return p.Theme
}

// ---------------------------------------------------------------- index

func (s *Server) benchOn(r *http.Request) bool {
	return s.opts.Bench || r.URL.Query().Get("bench") == "1"
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p := s.Prefs()
	bench := s.benchOn(r)
	q := p.Query()
	startPage, scrollRow := 0, -1
	selected := int64(0)
	if p.SelectedID != 0 && !bench {
		row, ok, err := core.FindRow(ctx, s.opts.Svc, q, p.SelectedID, 2000)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			startPage, scrollRow, selected = row/views.PageSize, row, p.SelectedID
		}
	}
	p.SelectedID = selected
	l, err := s.listVM(ctx, p, startPage, scrollRow)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	form, _ := s.formFor(ctx, selected)
	theme := s.theme(r, p)
	vm := views.PageVM{
		Theme:          theme,
		View:           p.View,
		Desktop:        s.Desktop(),
		NativeMenu:     s.opts.NativeMenu,
		Bench:          bench,
		ProcessStartMs: core.ProcessStart.UnixMilli(),
		Query:          q,
		Filters:        s.filterVM(p, l.Page.Total),
		List:           l,
		Form:           form,
		Host:           s.host,
		GoVersion:      runtime.Version(),
		BuildMarker:    BuildMarker,
		UIVersion:      FrameworkVersion,
	}
	vm.Filters.Theme = theme
	render(w, r, views.Page(vm))
}

// ---------------------------------------------------------------- list handlers

// queryFromForm reads search, cat and status from a posted form.
func queryFromForm(r *http.Request, base core.Prefs) core.Prefs {
	base.Search = r.FormValue("search")
	var cats []core.Category
	for _, v := range r.Form["cat"] {
		for _, name := range strings.Split(v, ",") {
			if c, err := core.ParseCategory(name); err == nil {
				cats = append(cats, c)
			}
		}
	}
	q := core.Query{Categories: cats}.Normalize()
	base.Categories = q.Categories
	base.Status = nil
	if st, err := core.ParseStatus(r.FormValue("status")); err == nil {
		base.Status = core.StatusPtr(st)
	}
	return base
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cur := s.Prefs()
	next := cur
	if r.Form.Has("search") || r.Form.Has("cat") || r.Form.Has("status") || r.Form.Has("filters") {
		next = queryFromForm(r, cur)
	}
	if sf := r.FormValue("sort"); sf != "" {
		f, err := core.ParseSortField(sf)
		if err == nil {
			if f == cur.SortBy {
				next.SortDesc = !cur.SortDesc
			} else {
				next.SortBy, next.SortDesc = f, false
			}
			switch r.FormValue("dir") {
			case "desc":
				next.SortDesc = true
			case "asc":
				next.SortDesc = false
			}
		}
	}
	l, err := s.listVM(ctx, next, 0, -1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	clearSel := false
	if next.SelectedID != 0 {
		found := false
		for _, it := range l.Page.Items {
			if it.ID == next.SelectedID {
				found = true
				break
			}
		}
		if !found {
			next.SelectedID, clearSel = 0, true
			l.SelectedID = 0
		}
	}
	s.update(func(p *core.Prefs) {
		p.Search, p.Categories, p.Status, p.SortBy, p.SortDesc, p.SelectedID = next.Search, next.Categories, next.Status, next.SortBy, next.SortDesc, next.SelectedID
	})
	f := s.filterVM(next, l.Page.Total)
	var out []templ.Component
	out = append(out, views.ListResponse(l, f))
	if clearSel {
		out = append(out, views.DetailForm(views.FormVM{}, true))
	}
	render(w, r, joinComponents(out...))
}

func joinComponents(cs ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, c := range cs {
			if err := c.Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	view := r.FormValue("view")
	if view != core.ViewTable && view != core.ViewCards {
		http.Error(w, "bad view", http.StatusBadRequest)
		return
	}
	p := s.update(func(p *core.Prefs) { p.View = view })
	startPage, scrollRow := 0, -1
	if p.SelectedID != 0 {
		if n, err := strconv.Atoi(r.FormValue("row")); err == nil && n >= 0 {
			startPage, scrollRow = n/views.PageSize, n
		} else if row, ok, _ := core.FindRow(ctx, s.opts.Svc, p.Query(), p.SelectedID, 2000); ok {
			startPage, scrollRow = row/views.PageSize, row
		}
	}
	l, err := s.listVM(ctx, p, startPage, scrollRow)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, r, views.ListResponse(l, s.filterVM(p, l.Page.Total)))
}

func (s *Server) handleGoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	p := s.Prefs()
	n, err := strconv.Atoi(strings.TrimSpace(r.FormValue("row")))
	if err != nil {
		n = 1
	}
	head, err := s.opts.Svc.Page(ctx, core.PageRequest{Query: p.Query(), Offset: 0, Limit: 1})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if head.Total == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	idx := max(0, min(head.Total-1, n-1))
	p.SelectedID = 0
	l, err := s.listVM(ctx, p, idx/views.PageSize, idx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	off := idx - l.PageIdx*views.PageSize
	if off < 0 || off >= len(l.Page.Items) {
		http.Error(w, "row out of range", http.StatusInternalServerError)
		return
	}
	it := l.Page.Items[off]
	l.SelectedID = it.ID
	l.ScrollRow = l.PageIdx*views.PageSize + off
	p = s.update(func(p *core.Prefs) { p.SelectedID = it.ID })
	form := views.FormVM{Item: &it, Values: views.ValuesFromItem(it), CSV: core.ItemCSVRow(it)}
	render(w, r, joinComponents(
		views.ListResponse(l, s.filterVM(p, l.Page.Total)),
		views.DetailForm(form, true),
	))
}

// handlePage returns one more page of the window (down, up or fill).
func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q, err := core.QueryFromValues(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pg, err := strconv.Atoi(r.URL.Query().Get("p"))
	if err != nil || pg < 0 {
		http.Error(w, "bad page", http.StatusBadRequest)
		return
	}
	view := r.URL.Query().Get("view")
	if view != core.ViewCards {
		view = core.ViewTable
	}
	t0 := time.Now()
	page, err := s.opts.Svc.Page(ctx, core.PageRequest{Query: q, Offset: pg * views.PageSize, Limit: views.PageSize})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sel := s.Prefs().SelectedID
	l := views.ListVM{View: view, Query: q, Page: page, PageIdx: pg, LastPage: lastPage(page.Total), SelectedID: sel, FetchMs: core.Ms(time.Since(t0))}
	dir := r.URL.Query().Get("dir")
	render(w, r, views.PageFragment(l, pg, dir, page.Items))
}

// ---------------------------------------------------------------- selection and form

func (s *Server) handleSelect(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	form, err := s.formFor(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.update(func(p *core.Prefs) { p.SelectedID = id })
	render(w, r, views.DetailForm(form, false))
}

func (s *Server) handleForm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	form, err := s.formFor(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	render(w, r, views.DetailForm(form, false))
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	old, err := s.opts.Svc.Get(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	it := old
	vals := views.FormValues{
		Name:     r.FormValue("name"),
		Category: r.FormValue("category"),
		Price:    r.FormValue("price"),
		Quantity: r.FormValue("quantity"),
		Status:   r.FormValue("status"),
		Rating:   r.FormValue("rating"),
		Restock:  strings.TrimSpace(r.FormValue("restock")),
		Notes:    r.FormValue("notes"),
		Featured: r.FormValue("featured") == "true",
	}
	errs := map[string]string{}
	it.Name = vals.Name
	if c, err := core.ParseCategory(vals.Category); err == nil {
		it.Category = c
	} else {
		errs["category"] = core.MsgCategory
	}
	if c, err := core.ParsePrice(vals.Price); err == nil {
		it.PriceCents = c
	} else {
		errs["priceCents"] = err.Error()
	}
	if n, err := strconv.ParseInt(strings.TrimSpace(vals.Quantity), 10, 32); err == nil {
		it.Quantity = int32(n)
	} else {
		errs["quantity"] = core.MsgQuantity
	}
	if st, err := core.ParseStatus(vals.Status); err == nil {
		it.Status = st
	} else {
		errs["status"] = core.MsgStatus
	}
	if f, err := strconv.ParseFloat(strings.TrimSpace(vals.Rating), 64); err == nil {
		it.Rating = core.RatingFromTenths(int(f*10 + 0.5))
	} else {
		errs["rating"] = core.MsgRating
	}
	it.Featured = vals.Featured
	it.RestockDate = nil
	if vals.Restock != "" {
		if d, err := core.ParseDate(vals.Restock); err == nil {
			it.RestockDate = &d
		} else {
			errs["restockDate"] = err.Error()
		}
	}
	it.Notes = vals.Notes
	if ve := core.ValidateItem(it); ve != nil {
		for _, fe := range ve.Errors {
			if _, bad := errs[fe.Field]; !bad {
				errs[fe.Field] = fe.Message
			}
		}
	}
	showErrors := func(errs map[string]string) {
		form := views.FormVM{Item: &old, Values: vals, Errors: errs, ForceDirty: true, CSV: core.ItemCSVRow(old)}
		render(w, r, views.DetailForm(form, false))
	}
	if len(errs) > 0 {
		showErrors(errs)
		return
	}
	saved, err := s.opts.Svc.Update(ctx, it)
	if ve := core.AsValidationError(err); ve != nil {
		for _, fe := range ve.Errors {
			errs[fe.Field] = fe.Message
		}
		showErrors(errs)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rowIdx, _ := strconv.Atoi(r.FormValue("rowidx"))
	form := views.FormVM{Item: &saved, Values: views.ValuesFromItem(saved), CSV: core.ItemCSVRow(saved)}
	w.Header().Set("HX-Trigger", `{"saved":{"id":`+strconv.FormatInt(saved.ID, 10)+`}}`)
	render(w, r, joinComponents(
		views.DetailForm(form, false),
		views.RefreshedItem(s.Prefs().View, saved, rowIdx),
	))
}

// ---------------------------------------------------------------- misc

func (s *Server) handleTheme(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	t := r.FormValue("theme")
	switch t {
	case core.ThemeSystem, core.ThemeLight, core.ThemeDark:
	default:
		http.Error(w, "bad theme", http.StatusBadRequest)
		return
	}
	s.update(func(p *core.Prefs) { p.Theme = t })
	http.SetCookie(w, &http.Cookie{Name: "gguix-theme", Value: t, Path: "/", MaxAge: 365 * 24 * 3600, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleWindow(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	wd, _ := strconv.Atoi(r.FormValue("w"))
	ht, _ := strconv.Atoi(r.FormValue("h"))
	if wd >= 1024 && ht >= 640 {
		s.update(func(p *core.Prefs) { p.WindowWidth, p.WindowHeight = wd, ht })
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHost(w http.ResponseWriter, r *http.Request) {
	s.exMu.Lock()
	has := s.lastExport != ""
	s.exMu.Unlock()
	render(w, r, views.HostBody(s.host, has))
}

func (s *Server) handlePalette(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, PaletteCSS())
}

// PaletteCSS renders the section 4.5.1 colors from the core palette as CSS
// for the custom-rendered elements (light by default, .dark for dark).
func PaletteCSS() string {
	var b strings.Builder
	emit := func(prefix string, pal core.Palette) {
		rule := func(sel, decl string) { fmt.Fprintf(&b, "%s%s{%s}\n", prefix, sel, decl) }
		for _, st := range core.AllStatuses {
			c := pal.Badge(st)
			rule(".status-"+st.String(), "background-color:"+c.Background+";color:"+c.Text+";border-color:transparent")
		}
		rule(".stock-track", "fill:"+pal.StockTrack)
		rule(".stock-fill", "fill:"+pal.StockFill)
		rule(".stock-fill-low", "fill:"+pal.StockFillLow)
		rule(".star-on", "fill:"+pal.StarFilled)
		rule(".star-off", "fill:"+pal.StarEmpty)
		rule(".featured-chip", "background-color:"+pal.Featured.Background+";color:"+pal.Featured.Text+";border-color:transparent")
		for _, c := range core.AllCategories {
			rule(".tile-"+c.String(), "background-color:"+pal.Tile(c)+";color:"+pal.TileText)
		}
	}
	emit("", core.Colors(false))
	emit(".dark ", core.Colors(true))
	return b.String()
}

var _ = json.Marshal
