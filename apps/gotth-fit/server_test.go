package gotth

import (
	"bufio"
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/prefs"
)

// Scenario results are counted so the run can end with the GGUIX_TESTS line.
var (
	tally struct {
		sync.Mutex
		passed, failed, skipped int
	}
)

func TestMain(m *testing.M) {
	code := m.Run()
	tally.Lock()
	fmt.Printf("GGUIX_TESTS passed=%d failed=%d not_feasible=%d\n", tally.passed, tally.failed, tally.skipped)
	tally.Unlock()
	os.Exit(code)
}

// scenario counts one T-scenario when the test ends.
func scenario(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		tally.Lock()
		defer tally.Unlock()
		switch {
		case t.Failed():
			tally.failed++
		case t.Skipped():
			tally.skipped++
		default:
			tally.passed++
		}
	})
}

// countingSvc counts Update calls.
type countingSvc struct {
	core.DataService
	updates atomic.Int32
}

func (c *countingSvc) Update(ctx context.Context, it core.Item) (core.Item, error) {
	c.updates.Add(1)
	return c.DataService.Update(ctx, it)
}

type fakeNative struct {
	savePath string
	notified chan string
}

func (f *fakeNative) SaveDialog(string) (string, bool, error) { return f.savePath, true, nil }
func (f *fakeNative) OpenDialog(string) (string, bool, error) { return "", false, nil }
func (f *fakeNative) Notify(title, body string) error {
	if f.notified != nil {
		f.notified <- title + "|" + body
	}
	return nil
}
func (f *fakeNative) CopyText(string) error { return nil }

type env struct {
	srv   *Server
	svc   *countingSvc
	store *prefs.FileStore
	dir   string
}

func newEnv(t *testing.T, rps int, native Native) *env {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GGUIX_PREFS_DIR", dir)
	svc := &countingSvc{DataService: memory.New(memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: rps, Seed: 1})}
	store := prefs.NewFileStoreAt(dir)
	srv, err := New(Options{Svc: svc, Prefs: store, Native: native, SaveDelay: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return &env{srv: srv, svc: svc, store: store, dir: dir}
}

func (e *env) do(method, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	r := httptest.NewRequest(method, path, body)
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	e.srv.Handler().ServeHTTP(w, r)
	return w
}

var rowRe = regexp.MustCompile(`<tr[^>]*data-id="(\d+)"[^>]*data-row="(\d+)"`)

func firstRows(body string, n int) []string {
	var ids []string
	for _, m := range rowRe.FindAllStringSubmatch(body, n) {
		ids = append(ids, m[1])
	}
	return ids
}

func cellText(body, col string, nth int) string {
	re := regexp.MustCompile(`data-col="` + col + `"[^>]*>([^<]*)<`)
	ms := re.FindAllStringSubmatch(body, nth+1)
	if len(ms) <= nth {
		return ""
	}
	return html.UnescapeString(strings.TrimSpace(ms[nth][1]))
}

func mustPage(t *testing.T, e *env, q core.Query, off, limit int) core.Page {
	t.Helper()
	p, err := e.svc.Page(context.Background(), core.PageRequest{Query: q, Offset: off, Limit: limit})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// T1: the app starts and loads the first rows.
func TestT1_StartsAndLoads(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	w := e.do("GET", "/", nil)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	ids := firstRows(body, 3)
	if strings.Join(ids, ",") != "1,2,3" {
		t.Fatalf("first rows %v", ids)
	}
	for i, want := range []string{"SKU-0000001", "SKU-0000002", "SKU-0000003"} {
		if got := cellText(body, "sku", i); got != want {
			t.Errorf("row %d sku %q want %q", i+1, got, want)
		}
	}
}

// T2: activating the Name header sorts ascending.
func TestT2_Sort(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	w := e.do("POST", "/query", url.Values{"search": {""}, "status": {""}, "sort": {"name"}})
	want := mustPage(t, e, core.Query{SortBy: core.SortName}, 0, 1).Items[0]
	body := w.Body.String()
	if got := cellText(body, "name", 0); got != want.Name {
		t.Fatalf("first name %q want %q", got, want.Name)
	}
	if !strings.Contains(body, "Name ▲") {
		t.Error("header lacks the ascending marker")
	}
}

// T3: searching updates the count in the status bar.
func TestT3_Search(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	w := e.do("POST", "/query", url.Values{"search": {"steel"}, "status": {""}})
	total := mustPage(t, e, core.Query{Search: "steel"}, 0, 1).Total
	if total == 0 {
		t.Fatal("no steel items in the dataset")
	}
	want := fmt.Sprintf("%s items</span>", core.FormatCount(total))
	if !strings.Contains(w.Body.String(), `id="sb-count"`) || !strings.Contains(w.Body.String(), want) {
		t.Fatalf("count not updated to %q", want)
	}
}

func formValue(body, name string) string {
	re := regexp.MustCompile(`name="` + name + `"[^>]*value="([^"]*)"|value="([^"]*)"[^>]*name="` + name + `"`)
	m := re.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	if m[1] != "" {
		return html.UnescapeString(m[1])
	}
	return html.UnescapeString(m[2])
}

// T4: selecting a row fills every form field.
func TestT4_Selection(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	it := mustPage(t, e, core.Query{}, 1, 1).Items[0] // item 2
	w := e.do("POST", "/select", url.Values{"id": {strconv.FormatInt(it.ID, 10)}})
	body := w.Body.String()
	check := func(label, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %q want %q", label, got, want)
		}
	}
	check("name", formValue(body, "name"), it.Name)
	check("price", formValue(body, "price"), core.FormatPrice(it.PriceCents))
	check("quantity", formValue(body, "quantity"), strconv.Itoa(int(it.Quantity)))
	check("rating", formValue(body, "rating"), core.FormatRating(it.Rating))
	check("restock", formValue(body, "restock"), func() string {
		if it.RestockDate != nil {
			return it.RestockDate.String()
		}
		return ""
	}())
	if !strings.Contains(body, `data-field="id">`+strconv.FormatInt(it.ID, 10)+`<`) || !strings.Contains(body, it.SKU) {
		t.Error("id or sku missing")
	}
	if !regexp.MustCompile(`<option[^>]*value="` + it.Category.String() + `"[^>]*selected`).MatchString(body) {
		t.Error("category not selected")
	}
	if !regexp.MustCompile(`id="f-status-` + it.Status.String() + `"[^>]*checked|checked[^>]*id="f-status-` + it.Status.String() + `"`).MatchString(body) {
		t.Error("status radio not checked")
	}
	hasFeatured := regexp.MustCompile(`<input type="checkbox"[^>]*checked[^>]*id="f-featured"|<input type="checkbox"[^>]*id="f-featured"[^>]*checked`).MatchString(body)
	if hasFeatured != it.Featured {
		t.Errorf("featured %v want %v", hasFeatured, it.Featured)
	}
	if !strings.Contains(body, html.EscapeString(it.Notes)) {
		t.Error("notes missing")
	}
	if !strings.Contains(body, core.FormatUpdated(it.UpdatedAt)) {
		t.Error("updated missing")
	}
	if e.srv.Prefs().SelectedID != it.ID {
		t.Error("selection not stored in prefs")
	}
}

func saveForm(it core.Item, name string) url.Values {
	v := url.Values{
		"id": {strconv.FormatInt(it.ID, 10)}, "name": {name}, "category": {it.Category.String()},
		"price": {core.FormatPrice(it.PriceCents)}, "quantity": {strconv.Itoa(int(it.Quantity))},
		"status": {it.Status.String()}, "rating": {core.FormatRating(it.Rating)}, "notes": {it.Notes},
	}
	if it.Featured {
		v.Set("featured", "true")
	}
	if it.RestockDate != nil {
		v.Set("restock", it.RestockDate.String())
	}
	return v
}

// T5: a name that is too short shows the message and does not call Update.
func TestT5_Validation(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	it := mustPage(t, e, core.Query{}, 0, 1).Items[0]
	w := e.do("POST", "/save", saveForm(it, "ab"))
	if !strings.Contains(w.Body.String(), "Name must be 3 to 80 characters.") {
		t.Fatal("validation message missing")
	}
	if !strings.Contains(w.Body.String(), `data-force-dirty="true"`) {
		t.Error("form should stay dirty")
	}
	if n := e.svc.updates.Load(); n != 0 {
		t.Fatalf("Update called %d times", n)
	}
}

// T6: a valid change is saved and the table row shows the new name.
func TestT6_Save(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	it := mustPage(t, e, core.Query{}, 0, 1).Items[0]
	f := saveForm(it, "Renamed Test Item")
	f.Set("rowidx", "0")
	w := e.do("POST", "/save", f)
	body := w.Body.String()
	if !regexp.MustCompile(`<tr[^>]*id="row-1"[^>]*>`).MatchString(body) || !strings.Contains(body, ">Renamed Test Item<") {
		t.Fatalf("refreshed row missing from response")
	}
	if !strings.Contains(body, `hx-swap-oob="true"`) {
		t.Error("row not swapped out of band")
	}
	rows := e.do("GET", "/page?p=0&dir=fill&view=table&sort=id&desc=0", nil).Body.String()
	if cellText(rows, "name", 0) != "Renamed Test Item" {
		t.Errorf("page shows %q", cellText(rows, "name", 0))
	}
}

// T7: choosing Dark applies the dark theme and is saved in the prefs.
func TestT7_ThemeAndPrefs(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	w := e.do("POST", "/theme", url.Values{"theme": {"dark"}})
	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d", w.Code)
	}
	cookies := w.Result().Cookies()
	page := e.do("GET", "/", nil, cookies...).Body.String()
	if !regexp.MustCompile(`<html[^>]*class="dark"`).MatchString(page) {
		t.Error("dark class missing on <html>")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		b, _ := os.ReadFile(filepath.Join(e.dir, "prefs.json"))
		if strings.Contains(string(b), `"theme":"dark"`) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(`prefs.json never contained "theme":"dark"`)
}

func waitExport(t *testing.T, e *env, state string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if e.srv.ExportState().State == state {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("export never reached %q (now %q)", state, e.srv.ExportState().State)
}

// T8: cancelling an export stops it, removes the partial file and re-enables
// the Export button (the button follows the SSE state).
func TestT8_ExportCancel(t *testing.T) {
	scenario(t)
	dest := filepath.Join(t.TempDir(), "gguix-export.csv")
	e := newEnv(t, 200, &fakeNative{savePath: dest})
	ts := httptest.NewServer(e.srv.Handler())
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/export/save", "", nil)
	if err != nil || resp.StatusCode != http.StatusAccepted {
		t.Fatalf("start: %v %v", err, resp)
	}
	waitExport(t, e, "running")
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("partial file should exist while running: %v", err)
	}
	if _, err := http.Post(ts.URL+"/export/cancel", "", nil); err != nil {
		t.Fatal(err)
	}
	waitExport(t, e, "cancelled")
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("partial file still exists: %v", err)
	}
	// The status area the browser sees no longer says running, so the
	// Export button is enabled again.
	r, err := http.Get(ts.URL + "/export/progress")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	sc := bufio.NewScanner(r.Body)
	var first string
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "data: ") && strings.Contains(sc.Text(), "export-marker") {
			first = sc.Text()
			break
		}
	}
	if !strings.Contains(first, `data-state="cancelled"`) {
		t.Fatalf("progress snapshot: %s", first)
	}
	// A second export can start again.
	resp2, err := http.Post(ts.URL+"/export/save", "", nil)
	if err != nil || resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("restart: %v %v", err, resp2)
	}
	http.Post(ts.URL+"/export/cancel", "", nil)
	waitExport(t, e, "cancelled")

	// Web target: the streamed download is aborted on cancel.
	e2 := newEnv(t, 200, nil)
	ts2 := httptest.NewServer(e2.srv.Handler())
	defer ts2.Close()
	done := make(chan error, 1)
	go func() {
		r, err := http.Get(ts2.URL + "/export")
		if err != nil {
			done <- err
			return
		}
		_, err = io.Copy(io.Discard, r.Body)
		done <- err
	}()
	waitExport(t, e2, "running")
	http.Post(ts2.URL+"/export/cancel", "", nil)
	select {
	case err := <-done:
		if err == nil {
			t.Error("web download ended cleanly after cancel; the browser would keep a truncated file")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("web export did not stop")
	}
	waitExport(t, e2, "cancelled")
}

// T9: selecting another row while dirty shows the confirm dialog. The guard
// runs in the browser, so the server-side test checks the pieces it needs and
// the browser test (tests/browser) drives the real dialog.
func TestT9_DirtyGuard(t *testing.T) {
	// The guard is browser code. See TestT9Browser, which counts as T9.
	if _, err := e2eHTML(t); err != nil {
		t.Fatal(err)
	}
}

// e2eHTML checks the server side of the guard: the confirm dialog markup.
func e2eHTML(t *testing.T) (string, error) {
	e := newEnv(t, 0, nil)
	body := e.do("GET", "/", nil).Body.String()
	for _, want := range []string{"Discard unsaved changes?", `id="confirm-discard"`, `id="confirm-cancel"`, `id="btn-save"`, `id="dirty-msg"`} {
		if !strings.Contains(body, want) {
			return body, fmt.Errorf("page lacks %s", want)
		}
	}
	return body, nil
}

// T10: custom cells use the section 4.5.1 colors and the core helpers.
func TestT10_CustomCells(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	body := e.do("GET", "/", nil).Body.String()
	it := mustPage(t, e, core.Query{}, 0, 1).Items[0]
	row := body[strings.Index(body, `id="row-1"`):]
	row = row[:strings.Index(row, "</tr>")]
	m := regexp.MustCompile(`data-slot="stock-bar" data-fraction="([0-9.]+)"`).FindStringSubmatch(row)
	if m == nil {
		t.Fatal("stock bar missing")
	}
	if f, _ := strconv.ParseFloat(m[1], 64); fmt.Sprintf("%.4f", f) != fmt.Sprintf("%.4f", core.StockFraction(it.Quantity)) {
		t.Errorf("fraction %v want %v", f, core.StockFraction(it.Quantity))
	}
	wantFill := "stock-fill"
	if core.StockLow(it.Quantity) {
		wantFill = "stock-fill-low"
	}
	if !strings.Contains(row, `class="`+wantFill+`"`) {
		t.Errorf("fill class %s missing", wantFill)
	}
	if !strings.Contains(row, `data-status="`+it.Status.String()+`"`) || !strings.Contains(row, "status-"+it.Status.String()) {
		t.Error("status badge missing")
	}
	full, half, empty := core.Stars(it.Rating)
	sm := regexp.MustCompile(`data-full="(\d)" data-half="(\d)" data-empty="(\d)"`).FindStringSubmatch(row)
	if sm == nil || sm[1] != strconv.Itoa(full) || sm[2] != strconv.Itoa(half) || sm[3] != strconv.Itoa(empty) {
		t.Errorf("stars %v want %d %d %d", sm, full, half, empty)
	}
	if got := strings.Count(row, `class="star-on"`) + strings.Count(row, `class="star-off"`); got != 5+half {
		t.Errorf("star shapes %d", got)
	}
	css := e.do("GET", "/palette.css", nil).Body.String()
	for _, dark := range []bool{false, true} {
		pal := core.Colors(dark)
		prefix := ""
		if dark {
			prefix = ".dark "
		}
		b := pal.Badge(it.Status)
		want := prefix + ".status-" + it.Status.String() + "{background-color:" + b.Background + ";color:" + b.Text
		if !strings.Contains(css, want) {
			t.Errorf("palette.css lacks %q", want)
		}
		if !strings.Contains(css, prefix+".stock-fill{fill:"+pal.StockFill) || !strings.Contains(css, prefix+".stock-track{fill:"+pal.StockTrack) ||
			!strings.Contains(css, prefix+".star-on{fill:"+pal.StarFilled) || !strings.Contains(css, prefix+".star-off{fill:"+pal.StarEmpty) {
			t.Errorf("palette.css lacks stock or star colors (dark=%v)", dark)
		}
	}
}

// T11: the cards view shows a card for item 1; selecting the second card
// fills the form.
func TestT11_Cards(t *testing.T) {
	scenario(t)
	e := newEnv(t, 0, nil)
	w := e.do("POST", "/view", url.Values{"view": {"cards"}})
	body := w.Body.String()
	items := mustPage(t, e, core.Query{}, 0, 2).Items
	card := body[strings.Index(body, `id="card-1"`):]
	card = card[:strings.Index(card, `id="card-2"`)]
	for _, want := range []string{html.EscapeString(items[0].Name), core.FormatPrice(items[0].PriceCents), `data-status="` + items[0].Status.String() + `"`} {
		if !strings.Contains(card, want) {
			t.Errorf("card 1 lacks %q", want)
		}
	}
	form := e.do("POST", "/select", url.Values{"id": {"2"}}).Body.String()
	if formValue(form, "name") != items[1].Name || !strings.Contains(form, `data-item-id="2"`) {
		t.Error("second card did not fill the form")
	}
	if e.srv.Prefs().View != core.ViewCards {
		t.Error("view not saved in prefs")
	}
}

// T12: the host panel and status bar show the host identity.
func TestT12_HostInfo(t *testing.T) {
	scenario(t)
	for _, desktop := range []bool{false, true} {
		var n Native
		want := host.Server(AppID)
		if desktop {
			n = &fakeNative{}
			want = host.Current(AppID)
		}
		e := newEnv(t, 0, n)
		panel := e.do("GET", "/host", nil).Body.String()
		if !strings.Contains(panel, `data-label="Source">`+string(want.Source)+`<`) {
			t.Errorf("desktop=%v: source %q missing", desktop, want.Source)
		}
		if !strings.Contains(panel, `data-label="Username">`+html.EscapeString(want.Username)+`<`) {
			t.Errorf("desktop=%v: username missing", desktop)
		}
		page := e.do("GET", "/", nil).Body.String()
		if !strings.Contains(page, `data-slot="identity">`+html.EscapeString(want.Username+"@"+want.Hostname)+`<`) {
			t.Errorf("desktop=%v: status bar identity missing", desktop)
		}
		if !desktop && !strings.Contains(panel, "Not available in the browser") {
			t.Error("web panel should disable reveal actions")
		}
	}
}

// TestT9Browser drives T9 through Chrome when node and Chrome are available.
func TestT9Browser(t *testing.T) {
	// counted as part of T9: replaces the skipped server-side placeholder
	node, err := execLookPath("node")
	chrome := os.Getenv("CHROME")
	if chrome == "" {
		chrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	}
	if _, serr := os.Stat(chrome); err != nil || serr != nil {
		scenario(t)
		t.Skip("node or Chrome not available")
	}
	scenario(t)
	e := newEnv(t, 0, nil)
	ts := httptest.NewServer(e.srv.Handler())
	defer ts.Close()
	out, err := execCommand(node, "tests/browser/t9.test.mjs", ts.URL)
	t.Log(out)
	if err != nil {
		t.Fatalf("browser test failed: %v", err)
	}
}
