<!-- @audience master -->
# GGUIX: Go GUI Framework Comparison Spec (master)

Version: 1.2 (draft for review)
Date: 2026-11-08
Status: DRAFT. Nothing is implemented until this spec is approved.

This is the **master** spec. Implementers never see it. Each implementer gets a generated brief (`briefs/<id>.md`) that contains only what that implementer needs to build their app: no evaluation factors, no metrics, no rubric, and no mention of the other frameworks. Appendix C defines how briefs are generated and checked.

**Audience tags.** A block between `<!-- @audience X -->` and `<!-- @end -->`, or a single line ending in `<!--@X-->`, is included only in the briefs for audience `X`:

- `master`: only in this document, never in any brief
- `brief`: in every brief
- a comma list of app IDs (for example `fyne,cogent`): only in those apps' briefs

Untagged text goes into every brief. In brief text, `{{ID}}` and `{{NAME}}` are replaced with the app ID and framework display name.

---

## 0. Experiment overview

### 0.1 Purpose

Compare five Go GUI approaches on equal terms:

| ID | Framework | UI approach |
|---|---|---|
| `fyne` | Fyne | Own-drawn widgets (OpenGL) |
| `cogent` | Cogent Core | Own-drawn widgets (WebGPU), Material 3 |
| `wails` | Wails v3 | Webview + Svelte 5 + Tailwind v4 + shadcn-svelte |
| `miqt` | MIQT | Native Qt 6 Widgets |
| `gotth` | GoTTH | Server-rendered HTML (templ + htmx + Tailwind + shadcn-templ); webview shell for desktop |

Evaluation factors (unweighted, no particular order):

1. Ease of bootstrapping the project and the iterative build loop
2. Runtime requirements (what the end user's machine needs)
3. Toolchain footprint (anything needed beyond the `go` tool)
4. Native desktop widgets and OS integration
5. Desktop and web reach from one codebase, measured as how much extra target-specific code it takes to support both (less is better, section 3.3 and Appendix A)
6. Core-constraint tolerance (does the web route survive a cgo dependency in the core)
7. Code clarity and complexity
8. Testability
9. Performance with a 1,000,000-row lazily paged table and a 1,000,000-item custom-rendered card list
10. Appeal of out-of-the-box styles
11. Native host access: identifying the running user and using host functionality, in desktop mode and in web-tech-in-desktop mode (section 5.11)
12. Custom rendering: custom-painted table cells and custom item views (sections 5.3 and 5.10)

### 0.2 Process and roles

1. **Main model (Opus 5.5)** writes this spec, then implements and tests the shared `core` module (section 4) and the `bench/` scripts (Appendix A) before any implementer starts.
2. **Brief generation**: `go run ./tools/mkbrief` produces `briefs/<id>.md` for each app and runs the leak check (Appendix C). The main model reads all five briefs before launch.
3. **Workspaces**: each implementer works in its own directory outside this repo, `~/src/gguix-work/<id>/`, containing only `BRIEF.md`, a copy of `core/`, `apps/<id>/` and `results/<id>/`. They are not told that other implementations exist.
4. **Five implementer subagents (Sonnet 5.5)** run in parallel, one per workspace, with the prompt in Appendix C.
5. **Collection**: when an implementer finishes, the main model copies `apps/<id>/` and `results/<id>/` back into this repo and checks that the workspace copy of `core/` is unchanged (`diff -r`).
6. **Measurement**: the main model runs `bench/` sequentially (one app at a time, nothing else running) to collect Appendix A metrics. Implementers only smoke-test the bench mode; their timings are not used.
7. **Blind judge subagent (Opus 5.5)**, which built none of the apps, scores the Appendix B rubric from the code, results files and metrics, with this master spec as context. Screenshots are given to the judge labeled A to E in a random order for the style score. The judge checks a sample of `shared` files for unmarked target-specific branches (section 3.3); any it finds count against code clarity.
8. **Main model** writes `results/REPORT.md`.

### 0.3 Repository layout (master)

```
go_gui_experiments/
  SPEC.md                     this master spec
  briefs/<id>.md              generated implementer briefs
  tools/mkbrief/              brief generator and leak check (module gguix/tools)
  core/                       module gguix/core   (main model writes)
  bench/                      measurement scripts   (main model writes)
  apps/<id>/                  copied back from each workspace
  results/<id>/               implementer results + metrics.json + screenshots/
  results/judge/  results/REPORT.md
```
<!-- @end -->
<!-- @audience brief -->
# GGUIX implementation brief: {{NAME}}

Date: 2026-11-08

You are implementing the GGUIX reference application with {{NAME}}. This brief is complete: build the app from this document, the `core` module in your workspace, and {{NAME}}'s own documentation. Where it says MUST, there is no discretion. Where it says "record", write the outcome into the results files in section 10, whether the attempt succeeded or not.
<!-- @end -->

---

## 1. Stack and pinned versions

| Item | Version |
|---|---|
| Go toolchain | `go1.27.3 darwin/arm64` (already installed) |
| Host | macOS 26.6.2, Apple Silicon, 10 cores, 64 GB RAM. Docker Desktop, Google Chrome, Node 26.10 / npm 11.19 and the Xcode Command Line Tools are installed. |
| Fyne | `fyne.io/fyne/v2 v2.8.2`; CLI `fyne.io/tools v1.7.4` | <!--@fyne-->
| Cogent Core | `cogentcore.org/core v0.5.27` (library and `core` CLI) | <!--@cogent-->
| Wails | `github.com/wailsapp/wails/v3 v3.1.0` (library and `wails3` CLI) | <!--@wails-->
| Frontend | `svelte 5.57.3`, `tailwindcss 4.3.5`, `@tailwindcss/vite 4.3.5`, `@sveltejs/vite-plugin-svelte 7.3.1`, `vite 8.3.6`, `shadcn-svelte 1.8.0` (CLI), `bits-ui 2.20.6`, `@tanstack/svelte-virtual 3.13.52`, `vitest 5.1.0`, `@testing-library/svelte` (latest at install time, record the version), `@playwright/test 1.65.0` | <!--@wails-->
| MIQT | `github.com/mappu/miqt v0.15.0`, Qt 6 from Homebrew (`qt` formula, record the version installed) | <!--@miqt-->
| templ | `github.com/a-h/templ v0.3.1100` | <!--@gotth-->
| Components | `github.com/axadrn/shadcn-templ/v2 v2.0.0` (formerly templUI) | <!--@gotth-->
| htmx | `htmx.org 2.0.11`, `htmx-ext-sse 2.2.4` (vendored files) | <!--@gotth-->
| Tailwind | standalone CLI `v4.3.5` | <!--@gotth-->
| Desktop shell | `github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6` | <!--@gotth-->

---

## 2. Rules

<!-- @audience brief -->
- **Workspace**: work only inside the workspace root you were given. Do not list, read or search outside it, except your Go module cache, `node_modules`, Homebrew prefixes and the documentation of the tools listed in section 1.
- **The `core/` directory is read-only.** If the core has a bug or a missing capability that blocks you, record it in `results/{{ID}}/friction.md` under category `core-issue`, work around it inside your app if possible, and continue.
<!-- @end -->
- Framework versions newer than your training data are pinned in section 1. Do not code from memory. Before using any framework API, verify it against the pinned version's source in the module cache (`go doc`, or read files under `$(go env GOMODCACHE)`) or its official docs.
- For npm packages, read the installed package's types in `node_modules` before using them. <!--@wails-->
- Use the framework's own recommended project bootstrap (its CLI template, if it has one) and record exactly what you ran.
- **Timebox rule**: if a required feature is blocked by a framework limitation after 3 genuinely different attempts or 45 minutes, whichever comes first, stop, record it as a `gap` in `results/{{ID}}/capabilities.md` with what you tried, implement the closest fallback described in this brief if one exists, and move on. Gaps are valid results, not failures.
- Do not add third-party dependencies beyond those listed in this brief and what the framework's own template generates, unless a requirement is impossible without one. Any addition MUST be recorded in `friction.md` with the reason.
- **Styling rule**: use the framework's default theme, default fonts, default widget look and default colors. No custom colors, custom fonts, custom-drawn widgets or theme overrides, except: the custom-rendered elements that sections 5.3 and 5.10 require (which use exactly the section 4.5.1 colors), the error text color (use the theme's own error/danger color), and layout spacing done with the framework's standard layout facilities.
- For styling, Tailwind utility classes are allowed for layout and spacing only. Use the component library's default theme with base color `neutral`. <!--@wails,gotth-->
- Keep a running log in `results/{{ID}}/friction.md` as you go (template in section 10.2), not reconstructed at the end.
- Write all results in plain English. Do not use em dashes.

---

## 3. Workspace layout and build interface

```
<workspace root>/
  BRIEF.md                 this document
  core/                    module gguix/core (read-only)
  apps/{{ID}}/             your app (module gguix/apps/{{ID}})
  results/{{ID}}/          your results files (section 10)
```

`apps/{{ID}}/go.mod` contains:

```
require gguix/core v0.0.0
replace gguix/core => ../../core
```

No `go.work` file is used.

### 3.1 Required files

| File | Purpose |
|---|---|
| `apps/{{ID}}/Makefile` | The make targets in section 3.2. |
| `apps/{{ID}}/README.md` | Prerequisites, exact bootstrap commands from a clean machine, how to run each target. |
| `apps/{{ID}}/marker.go` | `const BuildMarker = "0"` in the app's main Go package; shown in the About dialog. |
| `apps/wails/frontend/src/lib/marker.ts` | `export const UIMarker = "0";` rendered in the About dialog. | <!--@wails-->
| `apps/gotth/views/marker.templ` | A templ component rendering the string `UIMarker 0` in the About dialog. | <!--@gotth-->

### 3.2 Make targets

Every target MUST exist. A target that your framework cannot support MUST print `UNSUPPORTED: <one-line reason>` and exit with status 3.

| Target | Meaning |
|---|---|
| `deps` | Install everything needed beyond Go and Homebrew themselves (CLI tools at pinned versions, packages, libraries). Idempotent. |
| `dev` | The fastest edit-run loop the framework offers (hot reload if available). |
| `build` | Desktop binary for the host at `apps/{{ID}}/bin/gguix-{{ID}}`. |
| `package` | macOS `.app` bundle at `apps/{{ID}}/bin/GGUIX-{{ID}}.app` using the framework's packaging tool. |
| `run` | Build and run the desktop app. |
| `web` | Build the web target with the in-process core into `apps/{{ID}}/bin/web/`, then serve it on `http://localhost:8090`. | <!--@fyne,cogent-->
| `web-remote` | Build the web target with the `remote` build tag (section 4.8), then serve it with the core server: `cd ../../core && go run ./cmd/server -static ../apps/{{ID}}/bin/web -addr :8090` (section 4.12). | <!--@fyne,cogent-->
| `web` | Build and run the server-mode web target on `http://localhost:8090` (section 6). | <!--@wails,gotth-->
| `web-remote` | Alias of `web` that first prints one `NOTE:` line explaining that the core already runs on the server. | <!--@wails,gotth-->
| `web` | Expected to be `UNSUPPORTED` (section 6). | <!--@miqt-->
| `web-remote` | Expected to be `UNSUPPORTED` (section 6). | <!--@miqt-->
| `test` | All automated tests. Exit non-zero on failure. Output ends with the line in section 8. |
| `bench` | Run the desktop app in bench mode (section 9). |
| `bench-web` | Serve the `web` target so a browser can open it with `?bench=1` (section 9). | <!--@fyne,cogent,wails,gotth-->
| `bench-web` | Expected to be `UNSUPPORTED`. | <!--@miqt-->
| `cgoprobe` | Rebuild every buildable target with `-tags cgoprobe` (section 4.10) and print one line per target `CGOPROBE <target> PASS\|FAIL <reason>`. Never exits non-zero just because a target fails. |
| `cross-windows` | Attempt a `windows/amd64` desktop build (section 7). |
| `cross-linux` | Attempt a `linux/amd64` desktop build (section 7). |
| `clean` | Remove `bin/` and generated files. |

### 3.3 Marking target-specific code

<!-- @audience master -->
*Master note:* this section is what lets `bench/loc.sh` compute the dual-target overhead (Appendix A). The brief presents it only as a code organization rule; implementers are not told it is measured.
<!-- @end -->

Keep target-specific code visibly separate, so that a reader can tell which code serves which target. Every line of app source (Go, build files and any other source in your app) MUST fall into exactly one category:

| Category | Meaning |
|---|---|
| `shared` | Used by both the desktop target and the web target(s) |
| `desktop` | Exists only for the desktop target |
| `web` | Exists only for the web target(s) |
| `bench` | Bench driver code (section 9) |
| `test` | Test code |

How to mark it, in order of preference:

1. **Whole files**: Go files restricted with a build constraint end in `_desktop.go` or `_web.go`. Non-Go files used by only one target live under a directory named `desktop/` or `web/`. Bench code lives in files ending `_bench.go` (or `.bench.ts`/`.bench.js` for script files) or under a `bench/` directory inside the app. Tests use the language's normal test file conventions (`_test.go`, `*.test.ts`, `*.spec.ts`).
2. **Inline blocks**: when target-specific logic sits inside a shared file (an `if` on the platform, a capability check, a fallback branch), wrap it in marker comments using the file's comment syntax:
   ```go
   // gguix:target web
   ...lines...
   // gguix:end
   ```
   (`desktop` likewise; in HTML-like templates use `<!-- gguix:target web -->` and `<!-- gguix:end -->`; in Makefiles `# gguix:target web` / `# gguix:end`). Markers MUST NOT nest.
3. Everything else counts as `shared`.

For your framework, the web target(s) are expected to be unsupported, so most code will be `shared` or `desktop`. Still mark any desktop-only code consistently. <!--@miqt-->

---

## 4. The `core` module (`gguix/core`)

<!-- @audience master -->
The main model writes this module, with tests, before implementers start.
<!-- @end -->
Use it as-is. All of `core` is pure Go and compiles for `GOOS=js GOARCH=wasm`, except files guarded by the `cgoprobe` tag. Read its package docs (`go doc gguix/core/...`) for details beyond this section.

### 4.1 Packages

| Package | Contents |
|---|---|
| `gguix/core` | Domain types, `DataService`, `PrefsStore`, validation, formatting, `Pager`, palette, metrics helpers |
| `gguix/core/memory` | In-process `DataService` (generated dataset) |
| `gguix/core/remote` | HTTP client implementing `DataService`, `PrefsStore` and host info | <!--@fyne,cogent-->
| `gguix/core/prefs` | `FileStore` (native) and `LocalStorageStore` (`js/wasm`) |
| `gguix/core/backend` | `Open(appID string) (core.DataService, core.PrefsStore, error)`: picks the implementation by build tag |
| `gguix/core/bench` | Bench script runner for Go UIs (section 9.2) |
| `gguix/core/host` | Host information and host actions (section 4.11) |
| `gguix/core/cmd/server` | HTTP server exposing the section 4.12 API and optional static files | <!--@fyne,cogent-->

### 4.2 Domain types

```go
package core

type Category uint8 // 0..7, displayed in this order:
// Electronics, Garden, Kitchen, Office, Outdoor, Sports, Tools, Toys
func (c Category) String() string
func ParseCategory(s string) (Category, error)
var AllCategories []Category

type Status uint8 // 0 Active, 1 Backordered, 2 Discontinued
func (s Status) String() string
func ParseStatus(s string) (Status, error)
var AllStatuses []Status

type Date struct{ Year int; Month time.Month; Day int } // civil date, no time zone
func (d Date) String() string        // "2026-03-07"
func ParseDate(s string) (Date, error)

type Item struct {
    ID          int64      // 1..N, read-only
    SKU         string     // "SKU-0000001" (7 digits, zero padded), read-only
    Name        string
    Category    Category
    PriceCents  int64
    Quantity    int32
    Status      Status
    Rating      float32    // 0.0..5.0, one decimal place
    Featured    bool
    RestockDate *Date      // optional
    UpdatedAt   time.Time  // UTC, read-only, set by the service on Update
    Notes       string
}
```

JSON encoding uses these field names: `id, sku, name, category, priceCents, quantity, status, rating, featured, restockDate, updatedAt, notes`. `category` and `status` encode as their display strings. `restockDate` is `"YYYY-MM-DD"` or `null`. `updatedAt` is RFC 3339 UTC.

### 4.3 Query and paging

```go
type SortField string
const (
    SortID SortField = "id"; SortSKU = "sku"; SortName = "name"; SortCategory = "category"
    SortPrice = "price"; SortQuantity = "quantity"; SortStatus = "status"
    SortRating = "rating"; SortUpdated = "updated"; SortFeatured = "featured"
)

type Query struct {
    Search     string     // case-insensitive substring match on Name OR SKU; trimmed; "" = all
    Categories []Category // empty = all; otherwise item.Category must be in the set
    Status     *Status    // nil = all
    SortBy     SortField  // "" means SortID
    Desc       bool
}

type PageRequest struct {
    Query  Query
    Offset int // 0-based row index into the filtered, sorted result
    Limit  int // 1..500
}

type Page struct {
    Offset   int
    Items    []Item // len <= Limit; fewer only at the end
    Total    int    // number of rows matching Query
    QueryKey string // stable string identifying Query (for caches)
}

type FieldError struct{ Field, Message string } // Field uses the JSON field name
type ValidationError struct{ Errors []FieldError }
func (e *ValidationError) Error() string

var ErrNotFound = errors.New("item not found")

type DataService interface {
    Page(ctx context.Context, req PageRequest) (Page, error)
    Get(ctx context.Context, id int64) (Item, error)
    // Update validates, sets UpdatedAt, stores the edit (in memory, process
    // lifetime) and returns the stored item. Returns *ValidationError if invalid.
    Update(ctx context.Context, it Item) (Item, error)
    // Export streams CSV (header + one row per matching item in Query order) to w.
    // progress is called at most every 100 ms and once at the end, from the
    // exporting goroutine. Returns rows written. Honors ctx cancellation within 50 ms.
    Export(ctx context.Context, q Query, w io.Writer, progress func(done, total int)) (int, error)
}

// FindRow returns the 0-based row index of item id within the first maxRows
// rows of q's result, fetching pages of 200 through svc. ok is false if absent.
func FindRow(ctx context.Context, svc DataService, q Query, id int64, maxRows int) (row int, ok bool, err error)
```

Sort order: by `SortBy`, ascending unless `Desc`, ties always broken by `ID` ascending. String sorts are by byte order of the display string. `featured` sorts false before true. Edits from `Update` immediately affect later `Page` results.

### 4.4 Dataset and simulated latency

- `memory.New(memory.Options{Rows, Latency, ExportRowsPerSec, Seed})`. `memory.DefaultOptions()` returns `Rows 1_000_000`, `Latency 50ms`, `ExportRowsPerSec 100_000` (a full export takes about 10 s), `Seed 1`. In an `Options` literal, `Latency` is used as given (0 means none), `Rows` 0 means 1,000,000, `ExportRowsPerSec` 0 means 100,000 and negative means unthrottled.
- Environment overrides read by `backend.Open`: `GGUIX_ROWS`, `GGUIX_LATENCY_MS`, `GGUIX_EXPORT_RPS`.
- Rows are generated deterministically from `(Seed, ID)`. Names look like `"Brushed Steel Lantern"`. Generated `Quantity` is in `0..1000` (edits may set up to 1,000,000). About half the rows have empty `Notes`. About 10% are `Featured`. Searching `steel` returns results.
- `Latency` is applied once per `Page`, `Get` and `Update` call, after the work is done, and respects `ctx`.
- The first sort by a given field builds an index and may take noticeably longer than later requests.

### 4.5 Validation and formatting

`core.ValidateItem(it Item) *ValidationError` (nil if valid). Rules, with exact messages:

| Field | Rule | Message |
|---|---|---|
| `name` | trimmed length 3..80 | `Name must be 3 to 80 characters.` |
| `priceCents` | 0..10_000_000 | `Price must be between $0.00 and $100,000.00.` |
| `quantity` | 0..1_000_000 | `Quantity must be between 0 and 1,000,000.` |
| `rating` | 0.0..5.0 in steps of 0.1 | `Rating must be between 0.0 and 5.0.` |
| `restockDate` | required when status is Backordered | `Restock date is required for backordered items.` |
| `quantity` | must be 0 when status is Discontinued | `Discontinued items must have quantity 0.` |
| `notes` | at most 500 characters | `Notes must be at most 500 characters.` |

Formatting helpers. The UI MUST use these (or, for script code, exactly reproduce their output from values the Go side provides): <!--@wails-->
Formatting helpers. The UI MUST use these: <!--@fyne,cogent,miqt,gotth-->

```go
func FormatPrice(cents int64) string            // "$1,234.56"
func ParsePrice(s string) (int64, error)        // accepts "1234.56", "$1,234.56", "12"; error text: "Enter a price like 12.34."
func FormatRating(r float32) string             // "4.3"
func FormatUpdated(t time.Time) string          // "2026-01-02 15:04" (UTC)
func FormatCount(n int) string                  // "1,000,000"
func ItemCSVRow(it Item) string                 // one CSV line, same columns as Export
func StockFraction(qty int32) float64           // min(qty, 1000) / 1000, for the stock bar
func StockLabel(qty int32) string               // "Stock 432"
func Stars(r float32) (full, half, empty int)   // rating rounded to the nearest 0.5; full+half+empty == 5
func CategoryInitial(c Category) string         // "E" for Electronics, etc.
func NotesPreview(notes string) string          // first line, at most 90 characters plus "…" if cut; "No notes" if empty
```

#### 4.5.1 Custom rendering colors

Custom-rendered elements (sections 5.3 and 5.10) use these exact colors. `core.Colors(dark bool) Palette` returns them as hex strings, with helpers `Badge(status)`, `StockFillFor(qty)` and `Tile(category)`; `core.HexColor(hex)` converts one to `color.RGBA`, and `core.StockLow(qty)` tells when to use the low-stock fill.

| Element | Light | Dark |
|---|---|---|
| Status Active badge background / text | `#DCFCE7` / `#166534` | `#14532D` / `#BBF7D0` |
| Status Backordered badge background / text | `#FEF3C7` / `#92400E` | `#78350F` / `#FDE68A` |
| Status Discontinued badge background / text | `#E5E7EB` / `#374151` | `#374151` / `#E5E7EB` |
| Stock bar track / fill (fill when fraction < 0.2) | `#E5E7EB` / `#2563EB` (`#DC2626`) | `#374151` / `#60A5FA` (`#F87171`) |
| Star filled / empty | `#F59E0B` / `#D1D5DB` | `#FBBF24` / `#4B5563` |
| Featured chip background / text | `#EDE9FE` / `#5B21B6` | `#4C1D95` / `#DDD6FE` |
| Category tile (Electronics, Garden, Kitchen, Office, Outdoor, Sports, Tools, Toys), with white initial text | `#2563EB #16A34A #EA580C #64748B #0D9488 #DC2626 #CA8A04 #DB2777` | same |

### 4.6 Pager

Your app MUST use `core.Pager` for the table and the cards view. <!--@fyne,cogent,miqt-->
Your frontend MUST implement a pager in TypeScript (`frontend/src/lib/pager.ts`) that follows the algorithm in 4.6.1 exactly, with the same options and defaults, calling the Go service's `Page` method. The Go `core.Pager` API below is the reference for its shape. <!--@wails-->
Your app does not use a client-side pager; section 6 describes the server-rendered windowing that replaces it. The API below is for reference only. <!--@gotth-->

```go
type PagerOptions struct {
    PageSize       int // 200
    MaxPages       int // 20  (LRU of cached pages per query)
    MaxInFlight    int // 2
    PrefetchMargin int // 50 rows
}
func NewPager(svc DataService, opts PagerOptions, onChange func()) *Pager
func (p *Pager) SetQuery(q Query)            // clears cache, bumps generation, fetches page 0
func (p *Pager) SetVisible(first, last int)  // inclusive row indexes; schedules fetches
func (p *Pager) Total() int                  // -1 until the first page of the current query arrives
func (p *Pager) Row(i int) (Item, bool)      // non-blocking; false = not loaded (show placeholder)
func (p *Pager) RangeLoaded(first, last int) bool // Total known and every row in the range loaded
func (p *Pager) Invalidate()                 // after Update: refetch visible pages (old data stays until new arrives), drop the rest
func (p *Pager) Stats() PagerStats           // LastFetch time.Duration, CachedPages int, InFlight int
```

`onChange` is called from a background goroutine whenever rows or `Total` change. The app MUST marshal UI updates onto its UI thread. <!--@fyne,cogent,miqt-->

#### 4.6.1 Pager algorithm

1. Page `k` covers rows `[k*PageSize, (k+1)*PageSize)`.
2. `SetVisible(first, last)` computes needed pages for `[first-PrefetchMargin, last+PrefetchMargin]` clamped to `[0, Total-1]`. Pages that intersect `[first, last]` are "visible", others "prefetch". While `Total` is unknown (after `SetQuery`), only the page containing `first` is requested.
3. Fetch order: visible pages before prefetch pages, each group in ascending distance from `first`. At most `MaxInFlight` requests are in flight. A page already cached or in flight is not requested again.
4. Every response carries the generation it was requested under. Responses for an older generation are discarded.
5. The cache is LRU by last `SetVisible` touch. When over `MaxPages`, evict least recently touched pages that are not visible.
6. `Row(i)` returns the cached item or "not loaded".
7. A failed request is retried after one second.
8. `Invalidate()` bumps the generation, marks visible pages stale (they keep serving `Row` until refetched) and drops all other pages.

### 4.7 Preferences

```go
type Prefs struct {
    Theme        string     // "light" | "dark" | "system"; default "system"
    SortBy       SortField  // default "id"
    SortDesc     bool
    Search       string
    Categories   []Category // empty = all
    Status       *Status    // nil = all
    SelectedID   int64      // 0 = none
    View         string     // "table" | "cards"; default "table"
    WindowWidth  int        // desktop only; default 1280
    WindowHeight int        // desktop only; default 800
}
type PrefsStore interface {
    Load() (Prefs, error) // missing file/key returns DefaultPrefs(), nil
    Save(Prefs) error
}
func DefaultPrefs() Prefs
```

- `prefs.NewMemoryStore()` is an in-memory store for tests.
- `prefs.FileStore` path: `os.UserConfigDir()/gguix/<appID>/prefs.json`. `GGUIX_PREFS_DIR` overrides the directory (tests and bench use a temp dir).
- `prefs.LocalStorageStore` (js/wasm): key `gguix/<appID>/prefs`. <!--@fyne,cogent-->
- Apps MUST use the core `PrefsStore`, not the framework's own preference API. Record in `capabilities.md` what the framework's native facility would have been.

### 4.8 Backend selection

`backend.Open(appID)`, with `appID` = `{{ID}}`:

- default build: `memory.New(...)` with env overrides, plus `prefs.FileStore` (native) or `prefs.LocalStorageStore` (js/wasm).
- build tag `remote`: `remote.New(baseURL)` for both services. On js/wasm `baseURL` is the page origin (same-origin, no CORS). On native, `GGUIX_REMOTE_URL` (required). <!--@fyne,cogent-->

### 4.9 Metrics and bench helpers

```go
// EmitMetric writes one line: GGUIX_METRIC {"event":"...", ...fields}
// to stdout (native) or console.log (js/wasm).
func EmitMetric(event string, fields map[string]any)
var ProcessStart time.Time // set in core's init()
// BenchEnabled reports bench mode: env GGUIX_BENCH=1 (native) or ?bench=1 in the page URL (js/wasm).
func BenchEnabled() bool
```

### 4.10 The cgo probe

- `core/checksum.go` (`//go:build !cgoprobe`): pure Go `checksum(b []byte) uint32`.
- `core/checksum_cgo.go` (`//go:build cgoprobe`): the same function implemented in C via cgo.
- `Export` calls `checksum` on every row and writes a trailing CSV comment line `# checksum <hex>`, so the cgo code is linked whenever `cgoprobe` is set.

`make cgoprobe` rebuilds each target with `-tags cgoprobe` and records, per target, whether it builds and runs. Do not try to make failing targets pass; report them.

`make cgoprobe` covers: the desktop build, `web`, and for `web-remote` both the WASM client (built without `cgoprobe`) and the server (`core/cmd/server` built with `cgoprobe`). Then run `web-remote` with that server and confirm the app works. <!--@fyne,cogent-->
`make cgoprobe` covers: the desktop build and the server-mode web build. <!--@wails-->
`make cgoprobe` covers: the desktop build. <!--@miqt-->
`make cgoprobe` covers: the web server and the desktop shell. <!--@gotth-->

<!-- @audience master -->
*Master note, expected results:* any target that compiles `core` to `js/wasm` with `cgoprobe` fails to build. Targets where the core runs natively (desktop builds, Wails server mode, GoTTH server, `core/cmd/server`) should pass.
<!-- @end -->

### 4.11 Host information and actions (`gguix/core/host`)

```go
type Source string
const (
    SourceNative      Source = "native"         // read by this process on the user's own machine
    SourceServer      Source = "server-process" // read by a server process (web targets)
    SourceUnavailable Source = "unavailable"
)

type Info struct {
    Source    Source
    Username  string // login name
    FullName  string // "" if the platform cannot provide it
    UID       string
    HomeDir   string
    Hostname  string
    OS        string // runtime.GOOS
    Arch      string // runtime.GOARCH
    PID       int
    ConfigDir string // the prefs directory in use
    Locale    string // from LANG / LC_ALL, "" if unset
    Notes     []string // anything degraded, e.g. "full name unavailable without cgo"
}

func Current(appID string) Info // never fails; degraded fields are "" and explained in Notes
func Server(appID string) Info  // same data, Source set to server-process (for server-side web targets)
func RevealInFileManager(path string) error // macOS: `open -R <path>`; Linux: xdg-open of the parent dir; Windows: explorer /select,
func OpenWithDefaultApp(path string) error  // macOS: `open <path>`
```

- On native builds `Current` uses `os/user`, `os.Hostname`, and the environment. Built without cgo, `os/user` on macOS may not return the full name; `Current` records that in `Notes`. This is a real result, not a bug to work around.
- On `js/wasm` without the `remote` tag, `Current` returns `Source: unavailable` with the note `browser sandbox: no host access`. <!--@fyne,cogent-->
- On js/wasm with the `remote` tag, `Current` fetches `GET /api/host` and returns the server's process user with `Source: server-process`. <!--@fyne,cogent-->
- Prefer your framework's own facility for revealing or opening files, if it has one, and fall back to the core helpers. Record which was used.


<!-- @audience fyne,cogent -->
### 4.12 Remote HTTP API (`core/cmd/server`, used by `web-remote`)

Provided by `core`. Your `web-remote` target serves your WASM build through it.

```
cd core && go run ./cmd/server -addr :8090 [-static <dir>] [-rows N] [-latency-ms N] [-export-rps N]
# or, from your app module: go run gguix/core/cmd/server ...
# cgo-backed core: add -tags cgoprobe
```

| Method and path | Request | Response |
|---|---|---|
| `GET /api/items` | query params `offset, limit, search, cat` (comma-separated category names), `status`, `sort`, `desc` (`1`/`0`) | `200` JSON `Page` (`{"offset","items","total","queryKey"}`) |
| `GET /api/items/{id}` | | `200` Item, `404` |
| `PUT /api/items/{id}` | JSON Item | `200` Item, `422` `{"errors":[{"field","message"}]}` |
| `GET /api/export` | same query params as items | `200 text/csv`, streamed, header `X-Total-Count` |
| `GET /api/prefs/{appID}` | | `200` Prefs JSON |
| `PUT /api/prefs/{appID}` | Prefs JSON | `204` |
| `GET /api/host?app={appID}` | | `200` host Info JSON with `source: "server-process"` |
| `GET /*` | | static files from `-static` (with `index.html` fallback), only if `-static` given |

The server applies `latency-ms` to item endpoints, so the remote client does not add its own. It serves `.wasm` as `application/wasm` and gzip-compresses responses when the client accepts it. The `gguix/core/remote` client already implements this API; you normally only need `backend.Open` with `-tags remote`.

<!-- @end -->

---

## 5. Application behavior

### 5.1 Window

- Title: `GGUIX - {{NAME}}`.
- Default size 1280 x 800, minimum 1024 x 640. On launch restore the saved size (desktop only).

### 5.2 Layout (top to bottom)

1. **Menu bar** (desktop, section 5.6).
2. **Toolbar row**, left to right:
   - Search text field, placeholder `Search name or SKU`, width about 280 px. Applies the query 250 ms after the last keystroke (debounce).
   - Category filter: a button labeled `Categories` when none are selected, or `Categories (n)` when n are selected. Clicking opens a popup/menu with 8 checkboxes in `AllCategories` order plus a `Clear` action. Changes apply immediately.
   - Status filter: dropdown with options `All statuses`, `Active`, `Backordered`, `Discontinued`.
   - Go-to-row: numeric field, placeholder `Go to row`, plus a `Go` button (Enter also triggers). Input is a 1-based row number in the current result; out-of-range values clamp. The current view scrolls so that row is the top visible row and selects it.
   - View switcher: a two-option segmented control (or two toggle buttons) `Table` | `Cards`. Both views show the same query result and share the selection.
   - Spacer.
   - `Export CSV` button.
   - Theme dropdown: `System`, `Light`, `Dark`.
3. **Main area**: horizontal split. Left: the table or the cards view (about 65% width). Right: detail form (about 35%). The divider is draggable if the framework offers a split container; otherwise fixed (record as a gap).
4. **Status bar** (single line, bottom):
   - Left: `<count> items` using `FormatCount(Total)`, then ` · rows <a>-<b>` for the visible 1-based range, then ` · last fetch <ms> ms · <n> pages cached` from the pager stats. <!--@fyne,cogent,wails,miqt-->
   - Left: `<count> items` using `FormatCount(Total)`, then ` · rows <a>-<b>` for the rows currently in the DOM window, then ` · last fetch <ms> ms` (server time for the last page request). <!--@gotth-->
   - Right, only while an export runs: a determinate progress bar (about 200 px), percentage text `NN%`, and a `Cancel` button.
   - Far right, always: `<Username>@<Hostname>` and a `Host` button (section 5.11).

### 5.3 Table

Columns, in order, with header text, starting width in px, alignment and cell content:

| Header | Width | Align | Cell |
|---|---|---|---|
| ID | 80 | right | `id` |
| SKU | 110 | left | `sku` |
| Name | 260 | left | `name` |
| Category | 110 | left | `category` |
| Price | 100 | right | `FormatPrice` |
| Qty | 80 | right | `quantity` |
| Stock | 120 | left | **custom**: stock bar (below) |
| Status | 120 | left | **custom**: status badge (below) |
| Rating | 110 | left | **custom**: 5 stars plus `FormatRating` text (below) |
| Updated | 150 | left | `FormatUpdated` |
| Featured | 80 | center | `✓` if true, empty if false |

**Custom cells.** Use the framework's most idiomatic custom-cell mechanism and record which one you used. Colors are from section 4.5.1, following the current light/dark theme.

- Expected mechanism: a custom widget or container of canvas objects returned by the table's cell-creation callback. <!--@fyne-->
- Expected mechanism: a custom child widget per cell, or a custom-rendered widget. <!--@cogent-->
- Expected mechanism: a Svelte component per custom cell, using inline SVG for the stars and the stock bar. <!--@wails-->
- Expected mechanism: a `QStyledItemDelegate` subclass with custom `paint` (and `sizeHint`), set on those columns. No per-cell widgets. <!--@miqt-->
- Expected mechanism: server-rendered HTML with inline SVG, as templ components. <!--@gotth-->

- **Stock bar**: a horizontal bar 80 px wide, 8 px tall, rounded ends, vertically centered: a track with a fill of width `StockFraction(quantity)`. The fill uses the low-stock color when the fraction is below 0.2. To its right, the quantity number in the default text style. Not interactive.
- **Status badge**: a rounded pill (fully rounded ends) with horizontal padding about 8 px, containing the status text in the badge text color on the badge background.
- **Stars**: five 12 px stars drawn as shapes or glyphs (`Stars(rating)`: full, half, empty, where a half star is the left half filled), followed by a space and `FormatRating`.
- Placeholder rows show `…` instead of the custom content.

Table behavior:

- Header row stays fixed while the body scrolls. Columns are resizable by dragging if the widget supports it (record).
- Clicking a header sorts by that field ascending; clicking the active sort header again toggles descending/ascending. The active header shows ` ▲` (ascending) or ` ▼` (descending) after its text, as part of the header label.
- Changing search, filters or sort resets the scroll to the top and reloads from the first page. The selection is kept only if the selected item is in the first page after the reload; otherwise cleared.
- The body is virtualized: only visible rows (plus the widget's own overscan) exist as widgets/DOM nodes. The scrollbar represents the full `Total` row count. <!--@fyne,cogent,wails,miqt-->
- The body is a windowed infinite scroll (section 6), not full virtualization. <!--@gotth-->
- A row whose data is not loaded shows `…` in every cell.
- Single selection by click. Up/Down arrow keys move the selection when the table has focus. The selected row is highlighted with the framework's default selection style.
- Row height: the framework's default for its table/list widget.
- Zero results: the table body shows the centered text `No items match.`

### 5.4 Detail form

Shown for the selected item. With no selection, every control is disabled and the form shows the text `Select an item to edit.` at the top.

| Label | Control | Notes |
|---|---|---|
| ID | read-only text | |
| SKU | read-only text | |
| Name | single-line text field | |
| Category | dropdown, 8 options | |
| Price | single-line text field | Shows `FormatPrice`; parsed with `ParsePrice` on Save |
| Quantity | integer spin box, 0..1,000,000, step 1 | If no spin box exists, a text field with integer validation (record gap) |
| Status | radio group, 3 options, horizontal | |
| Rating | slider 0.0..5.0, step 0.1, with a value label `FormatRating` to its right | |
| Featured | checkbox | |
| Restock date | date picker plus a `Clear` button | If no date picker exists, a text field accepting `YYYY-MM-DD` validated with `ParseDate` (record gap) |
| Updated | read-only text | |
| Notes | multi-line text area, 4 visible lines, wraps | |

Buttons below the form, left to right: `Save` (primary/default style), `Revert`, `Copy Row`, `Import Notes…`.

Behavior:

- **Dirty state**: any change from the loaded item makes the form dirty. While dirty, show the text `Unsaved changes` next to the buttons and enable `Save` and `Revert`; otherwise both are disabled.
- **Save**: parse fields (price parse errors use the `ParsePrice` message under the `priceCents` field), run `core.ValidateItem`, and if valid call `DataService.Update`. On `*ValidationError`, show each message directly under its field in the theme's error color, and keep the form dirty. On success, clear errors, reload the form from the returned item, refresh the cached data so the table row and card show the new values.
- **Revert**: show the confirm dialog `Discard unsaved changes?` with buttons `Discard` and `Cancel`. On Discard, reload the form from the item.
- **Selecting a different row while dirty**: show the same confirm dialog. On Cancel, keep the current selection and edits.
- **Copy Row**: copy `core.ItemCSVRow(item)` of the saved item to the system clipboard; show a transient message `Row copied` (status bar text for 2 s is acceptable).
- **Import Notes…**: open a native file-open dialog filtered to `.txt`. Read the file (UTF-8), put at most the first 500 characters into the Notes field (form becomes dirty). Files over 64 KB are rejected with an error dialog `File is too large.`

### 5.5 Export (background job)

- Triggered by the `Export CSV` button or File > Export CSV….
- Desktop: open a native save dialog with default file name `gguix-export.csv`. If cancelled, do nothing.
- Run `DataService.Export` for the current query in the background, writing to the chosen file.
- While running: the status bar shows progress (section 5.2), the `Export CSV` button and menu item are disabled, and the whole UI remains usable (scrolling, editing).
- `Cancel` cancels the context; the partial file is deleted; the status bar shows `Export cancelled` for 2 s.
- On completion: send an OS notification titled `Export complete` with body `<FormatCount(n)> rows written`. If the framework cannot send OS notifications, show an in-app message with the same text (record gap).
- On error: an error dialog with the error text.
- Web targets: instead of the save dialog, the export is delivered as a browser download named `gguix-export.csv`. With the in-process core (WASM), build the file in memory and trigger a download; with `web-remote`, stream it from the server's export endpoint. Progress and cancel are still required. If the framework cannot trigger a download from WASM, record a gap. <!--@fyne,cogent-->
- Web target: instead of the save dialog, the export streams from the server as a browser download named `gguix-export.csv`. Progress and cancel are still required. <!--@wails,gotth-->

### 5.6 Menus and shortcuts (desktop)

Use the platform's native menu bar where the framework supports it. On macOS `Cmd`, elsewhere `Ctrl`.

- **App menu (macOS) / Help menu (others)**: `About GGUIX` opens the About dialog. `Quit` (`Cmd+Q`).
- **File**: `Export CSV…` (`Cmd+E`), `Import Notes…` (`Cmd+O`), `Reveal Last Export`.
- **Edit**: the standard Cut/Copy/Paste/Select All for text fields (the framework's default edit menu if it provides one), plus `Copy Row` (`Cmd+Shift+C`).
- **View**: `Table` (`Cmd+1`), `Cards` (`Cmd+2`) as radio items; `Theme` submenu with radio items `System`, `Light`, `Dark`; `Reload` (`Cmd+R`) reloads the current query from the first page; `Focus Search` (`Cmd+F`).
- **Help**: `Host Info…` (`Cmd+I`).

About dialog content, one item per line: `GGUIX - {{NAME}}`, framework version, Go version (`runtime.Version()`), `BuildMarker <value>`. <!--@fyne,cogent,miqt-->
About dialog content, one item per line: `GGUIX - {{NAME}}`, framework version, Go version (`runtime.Version()`), `BuildMarker <value>`, `UIMarker <value>`. <!--@wails,gotth-->

### 5.7 Theme

- `System` follows the OS appearance and updates live when the OS changes, if the framework supports it (record).
- The theme dropdown and the View > Theme menu stay in sync.
- Theme changes apply without restart.

### 5.8 Preferences behavior

- Saved through the core `PrefsStore` 500 ms after the last change (debounce) and on quit.
- Saved fields: theme, view (table or cards), sort field and direction, search text, category filter, status filter, selected item ID, window size (desktop).
- On launch, restore all of them before the first query. Then call `core.FindRow(ctx, svc, query, SelectedID, 2000)`. If it returns a row index, scroll so that row is the top visible row and select it; otherwise leave nothing selected.
- From the frontend, call `FindRow` through the Go service (section 6). <!--@wails-->
- `FindRow` runs on the server when rendering the initial page. <!--@gotth-->

### 5.9 Threading

All UI mutation happens on the framework's UI thread using its documented mechanism. Record which mechanism you used in `capabilities.md`.

### 5.10 Cards view (custom item rendering)

A virtualized vertical list showing the same query result as the table, one custom-rendered card per item, with the same paging as the table. The list must be built from the framework's list/collection widget with a custom item, not by drawing everything onto one big canvas.

- **Card size**: full width of the list, fixed height 96 px, 8 px gap between cards, 12 px inner padding, rounded corners, a 1 px border using the theme's default border/separator color, the theme's default surface/background.
- **Card layout**, left to right:
  1. **Category tile**: 48 x 48 rounded square in the category color (section 4.5.1) with `CategoryInitial` centered in white, bold.
  2. **Middle column** (fills the space): line 1 the name in bold; line 2 `<SKU> · <Category>` in the theme's muted/secondary text; line 3 `NotesPreview(notes)` in muted text, single line, clipped with an ellipsis; line 4 a chip row with the status badge (section 5.3) and, if featured, a `Featured` chip in the featured colors.
  3. **Right column** (about 160 px, right-aligned): the price in a larger bold font (`FormatPrice`), the five stars plus rating text, and the stock bar with `StockLabel(quantity)` to its left.
- **Selected card**: border becomes 2 px in the theme's primary/accent color. Clicking selects (same rules as the table, including the dirty guard). Up/Down arrows move the selection when the list has focus.
- **Placeholder card**: the same frame with the text `Loading…` where the name goes and nothing else.
- **Zero results**: centered `No items match.`
- Changing search, filters or sort resets the list to the top. Go-to-row works in this view too (the target item becomes the top card).
- Switching views keeps the selection and scrolls the new view so the selected item is visible (top position), or to the top if none is selected.
- The current view is saved in prefs.

Expected mechanism (use it unless the pinned version offers a clearly more idiomatic one; record what you used):

- `widget.List` driven by the pager, with a custom widget (`widget.BaseWidget` plus renderer) or a container of canvas objects as the item. <!--@fyne-->
- Cogent Core's virtualized list widget with a custom frame of child widgets, or a custom-rendered widget, as the item. <!--@cogent-->
- `@tanstack/svelte-virtual` with a `Card.svelte` component using the component library's card and badge styling plus inline SVG for stars and the stock bar. <!--@wails-->
- `QListView` with a list model backed by the pager and a `QStyledItemDelegate` subclass with custom `paint` and `sizeHint`. No per-item widgets. <!--@miqt-->
- The same windowed infinite scroll as the table (section 6), with a templ card component using the component library's card and badge styling plus inline SVG. <!--@gotth-->

### 5.11 Host panel (native host access)

A dialog opened from Help > `Host Info…` (`Cmd+I`) and from the `Host` button in the status bar. It shows the host info as a two-column label/value list in this order: Source, Username, Full name, UID, Home directory, Hostname, OS / Arch, PID, Config directory, Locale, then each Note on its own line. Empty values show `(unavailable)`.

The status bar shows `<Username>@<Hostname>` (or `unknown user` when unavailable) just left of the `Host` button, so the identity is visible at all times.

Host actions in the same dialog:

- `Reveal Config Folder`: reveals the prefs directory in Finder.
- `Reveal Last Export`: reveals the most recently exported file in Finder; disabled until an export completes in this session. Also available as File > `Reveal Last Export`.

Required source per target (record the actual result in `capabilities.md`):

| Target | Source and actions |
|---|---|
| Desktop app | `native` from `host.Current`; actions enabled | <!--@fyne,cogent,miqt-->
| `web` (WASM, in-process core) | `unavailable`; actions disabled with the label or tooltip `Not available in the browser` | <!--@fyne,cogent-->
| `web-remote` (WASM, remote core) | `server-process` from `GET /api/host`; actions disabled (they would act on the server) | <!--@fyne,cogent-->
| Desktop app (webview UI) | `native`: the Go side calls `host.Current` and passes the result to the UI; actions run in Go on the local machine | <!--@wails,gotth-->
| `web` (server mode, browser UI) | `server-process` from `host.Server`; actions disabled with the label or tooltip `Not available in the browser` | <!--@wails,gotth-->

The UI MUST show the Source row, so a server-process identity is never presented as if it were the browser user.

---

## 6. Framework-specific requirements

Anything not stated here is your choice within the framework's recommended practice; record notable choices in `friction.md`.

<!-- @audience fyne -->
### 6.1 Fyne

- Bootstrap: plain `go mod init gguix/apps/fyne` plus `fyne.io/fyne/v2 v2.8.2`. CLI: `go install fyne.io/tools/cmd/fyne@v1.7.4`.
- Table: `widget.Table` with its header row support. Menus: `fyne.MainMenu`. Dialogs: the `dialog` package, and the native file dialogs if the version offers them (record whether dialogs are native or Fyne-drawn). Notifications: `App.SendNotification`.
- Packaging: `fyne package` for the `.app`.
- Web: Fyne's web/WASM target via the `fyne` CLI (`fyne serve` or `fyne package` with the web or wasm target; verify the exact flags with `fyne help` for v1.7.4 and record them). `web-remote` builds the same with `-tags remote` and serves the output with `core/cmd/server -static`.
- Tests: `fyne.io/fyne/v2/test`.
<!-- @end -->

<!-- @audience cogent -->
### 6.2 Cogent Core

- Bootstrap: `go install cogentcore.org/core/cmd/core@v0.5.27`, project per the Cogent Core docs.
- Table: Cogent Core's table/list widget backed by `core.Pager`. Do not materialize 1,000,000 items in a slice; if the widget can only bind to a slice, record it and implement the closest virtualized approach the framework allows.
- Packaging: `core pack` (or the documented equivalent).
- Web: the `core` CLI's web target (`core build web` / `core run web`; if v0.5.27 names the platform `wasm`, use that; record the exact command). `web-remote` builds the same with `-tags remote` and serves the output with `core/cmd/server -static`.
- Tests: Cogent Core's testing utilities (for example render assertions) plus plain Go tests.
<!-- @end -->

<!-- @audience wails -->
### 6.3 Wails v3

- Bootstrap: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`, then `wails3 init` with the Svelte TypeScript template (list templates with `wails3 init -l`; record the name used). Then add Tailwind v4 via `@tailwindcss/vite`, then `npx shadcn-svelte@1.8.0 init` with base color `neutral`, then add components: `button input select checkbox radio-group slider label textarea dialog alert-dialog dropdown-menu popover progress calendar table card badge sonner`. Pin npm versions from section 1 in `package.json` and commit `package-lock.json`.
- Backend: one Wails service `ItemService` wrapping the core `DataService` and `PrefsStore`, with methods `Page, Get, Update, FindRow, LoadPrefs, SavePrefs, StartExport, CancelExport, CopyRow, HostInfo, RevealConfigFolder, RevealLastExport, Format` (`Format` or pre-formatted fields on a view type: your choice, but displayed text MUST equal the core helpers' output). In server mode `HostInfo` returns `host.Server` and the reveal methods return an error. Export progress goes to the frontend through Wails events: `export:progress {done,total}`, `export:done {rows}`, `export:error {message}`, `export:cancelled`.
- Pager: `frontend/src/lib/pager.ts` per section 4.6.1.
- Table: shadcn-svelte table styling with `@tanstack/svelte-virtual` for virtualization. Scroll height = `Total * rowHeight`.
- Native: Wails v3 application menu, native dialogs, clipboard and notifications services.
- Packaging: `wails3 package` (or the template's task); record what it produced.
- Web: Wails v3 server mode (`-tags server`, the template's `build:server` task if present). The `web` target builds it and runs it with `WAILS_SERVER_PORT=8090`. Native-only features in server mode (dialogs, notifications, menus) fall back to browser equivalents; record each.
- Tests: Go tests for `ItemService`; Vitest plus `@testing-library/svelte` for components and `pager.ts`; optionally Playwright against server mode.
<!-- @end -->

<!-- @audience miqt -->
### 6.4 MIQT

- Prerequisites: `brew install qt pkgconf` (Qt 6). Bootstrap: `go get github.com/mappu/miqt@v0.15.0`, import `github.com/mappu/miqt/qt6`. Dynamic linking against Homebrew Qt.
- Table: `QTableView` with a custom `QAbstractTableModel` backed by `core.Pager` (`rowCount` returns `Total`; `data` returns placeholders for unloaded rows; emit `dataChanged` on pager changes). Use `qt6/mainthread` to marshal pager callbacks onto the Qt main thread.
- Widgets: `QLineEdit`, `QComboBox`, `QSpinBox`, `QRadioButton` in a `QButtonGroup`, `QSlider`, `QCheckBox`, `QDateEdit` (with calendar popup), `QPlainTextEdit`, `QSplitter`, `QProgressBar`, `QMenuBar`, `QFileDialog`, `QMessageBox`, `QClipboard`, `QDesktopServices` for reveal/open where suitable. Notifications: `QSystemTrayIcon::showMessage` if available in the bindings, otherwise an in-app message (record).
- Theme: Qt follows the macOS appearance by default. Light/Dark override via `QStyleHints.setColorScheme` if available in the bound Qt version; otherwise record a gap and implement System only.
- Packaging: build a `.app` bundle; attempt `macdeployqt` (from Homebrew Qt) to make it self-contained and record the result and the size both before and after.
- Web: MIQT has no web target. `web`, `web-remote` and `bench-web` print `UNSUPPORTED`. If you find a supported route in MIQT's docs, record it but do not implement it.
- Cross builds: MIQT documents `miqt-docker` (Docker is installed). Try it for both targets.
- Tests: Go tests with a headless `QApplication` (`QT_QPA_PLATFORM=offscreen`) exercising the model and widgets; use `qt6/qtest` only if bound.
- The first compile of the Qt bindings takes a long time. This is normal; do not try to work around it.
<!-- @end -->

<!-- @audience gotth -->
### 6.5 GoTTH (Go + templ + Tailwind + htmx)

- Prerequisites (Go-installed or standalone binaries, no npm): `go install github.com/a-h/templ/cmd/templ@v0.3.1100`; the shadcn-templ v2 CLI (`go install github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@v2.0.0`; verify the path and commands with its `--help`); the Tailwind standalone CLI `v4.3.5` downloaded into `apps/gotth/bin/tools/` by `make deps`; htmx `2.0.11` and `htmx-ext-sse 2.2.4` vendored as files under `static/`. If any of these turns out to require npm, record it in `toolchain.md` and `friction.md` and use npm only for that piece.
- Server: Go `net/http` (standard library router) in `apps/gotth/cmd/web`, using the core `DataService` and `PrefsStore` in-process. Prefs are stored server side (one user).
- Components: shadcn-templ components installed into the project with its CLI. Use them for buttons, inputs, selects, checkboxes, radio groups, sliders, dialogs, popovers, progress, date picker, toasts, cards, badges and table styling.
- Table and cards windowing: rows are rendered on the server in pages of 200 using `DataService.Page`. Inside a fixed-height scroll container, a bottom sentinel row with `hx-trigger="revealed"` loads the next page and a top sentinel loads the previous page when scrolled up. Keep at most 5 pages (1,000 rows or cards) in the DOM: when a 6th page is added, remove the page furthest from the viewport and replace it with a spacer of the same height so the scroll position does not jump. Go-to-row and sort/filter changes replace the whole window, starting at the target page. The scrollbar represents only the loaded window; record this.
- Selection, form load and save: htmx requests returning form fragments. Validation errors are rendered by the server. The dirty-state check and confirm dialogs run in the browser (a small inline script is acceptable).
- Export: `GET /export` streams the CSV as a download; progress is pushed over SSE (`/export/progress`) with the htmx SSE extension; cancel is `POST /export/cancel`.
- Browser features: Clipboard API, Notification API (request permission on first export), `<input type="file">` for Import Notes, keyboard shortcuts via a small script. There is no menu bar in the browser; record it.
- Theme: the shadcn-templ `dark` class on `<html>`. The theme is also stored in a cookie so the server renders the right class on first paint (no flash). `System` uses `prefers-color-scheme`.
- Desktop: `apps/gotth/cmd/desktop` starts the same server in the same process on a random free `127.0.0.1` port with desktop mode enabled, and opens it in a `webview_go` window (1280 x 800). In desktop mode the host panel uses `host.Current` (`Source: native`) and the reveal actions run on the local machine; in `cmd/web` it uses `host.Server` and the actions are disabled. The desktop server MUST listen only on `127.0.0.1`. Record which native features are missing in this shell (menus, native dialogs, notifications) and the fallback used for each.
- `web` runs `cmd/web` on port 8090. `package` builds `cmd/desktop` into a `.app`.
- Tests: templ components rendered to strings in Go tests, handlers tested with `net/http/httptest`, optionally Playwright.
<!-- @end -->

---

## 7. Cross-compilation attempts

For `cross-windows` and `cross-linux`, try the framework's documented cross-compilation route first. Docker is available. Timebox 30 minutes per target. Record in `capabilities.md`: route used, extra tools needed, PASS/FAIL, output size, and the error if it failed. A produced binary counts as PASS; running it on Windows or Linux is not required.

- Suggested routes: `fyne-cross` (Docker) or the `fyne` CLI's cross-target options. <!--@fyne-->
- Suggested route: the `core` CLI's build command with a target platform. <!--@cogent-->
- Suggested route: the cross-platform build tasks generated by the `wails3` template. <!--@wails-->
- Suggested route: `miqt-docker` (for example `miqt-docker win64-qt6-static -windows-build`, and a Linux Qt 6 image). <!--@miqt-->
- Suggested route: plain `GOOS=... GOARCH=... go build` for `cmd/web`; `cmd/desktop` needs cgo for the webview, so try a Docker-based cross toolchain and record the result. <!--@gotth-->

---

## 8. Testing requirements

Every app MUST attempt these scenarios as automated tests run by `make test`, using `memory.New(memory.Options{Rows: 1000, Latency: 0})` and a temp `GGUIX_PREFS_DIR` (for T8 also set `ExportRowsPerSec: 200` so the export runs long enough to cancel):

| ID | Scenario | Pass condition |
|---|---|---|
| T1 | App starts and loads | First visible rows show IDs 1, 2, 3 and the correct SKU text |
| T2 | Sort | Activating the Name header sorts ascending; the first row's name equals the core's first row for `SortName` |
| T3 | Search | Setting search to `steel` updates the count in the status bar to the core's `Total` for that query |
| T4 | Selection | Selecting a row fills every form field with that item's values |
| T5 | Validation | Entering name `ab` and saving shows `Name must be 3 to 80 characters.` and does not call Update |
| T6 | Save | A valid name change saves and the table row shows the new name |
| T7 | Theme and prefs | Choosing Dark applies the dark theme and the saved prefs contain `"theme":"dark"` |
| T8 | Export cancel | Starting an export then cancelling stops it, removes the partial file and re-enables the Export button |
| T9 | Dirty guard | Selecting another row while dirty shows the confirm dialog |
| T10 | Custom cells | For row 1, the Stock cell's bar fraction equals `StockFraction(quantity)`, the Status badge shows the status text with the section 4.5.1 colors for the current theme, and the star counts equal `Stars(rating)` |
| T11 | Cards view | Switching to Cards shows a card for item 1 with its name, `FormatPrice` text and status badge; clicking the second card selects item 2 and fills the form |
| T12 | Host info | The Host panel shows `Source` and `Username` equal to the expected host info for the target under test, and the status bar shows `<Username>@<Hostname>` |

Drive the UI through the framework's testing tools, through the app's own presentation layer, or through a browser driver, in that order of preference. A scenario that cannot be automated is marked `not feasible` in `capabilities.md` with the reason. `make test` output must end with the line `GGUIX_TESTS passed=<n> failed=<n> not_feasible=<n>`.

---

## 9. Bench mode

### 9.1 Trigger

Desktop: env `GGUIX_BENCH=1`. Web: URL query `?bench=1`. In bench mode the app uses the default dataset (1,000,000 rows, 50 ms latency), the prefs dir from `GGUIX_PREFS_DIR` (set by whoever runs it), skips restoring the selection, and exports to the path in `GGUIX_BENCH_EXPORT` (desktop) without a dialog (web: no download, the CSV is discarded).

### 9.2 Script

A **UI tick probe** runs during steps 3, 4 and 7: every 16 ms, schedule a no-op on the UI thread and record the delay until it runs (Go UIs: from a goroutine through the framework's UI-thread mechanism; browser UIs: `setInterval` drift). `max_gap_ms` is the maximum delay in the phase, `p95_gap_ms` the 95th percentile.

"Loaded" means every row (or card) in the visible range has data (no `…` / `Loading…`) and the UI refresh call for it has been made.

The bench starts in the Table view. Each step emits one `GGUIX_METRIC` line:

1. `ready`: from process start (desktop) or `performance.timeOrigin` (web) until the window is shown and the first visible page is loaded. Fields: `ms`.
2. `jump`: go to row 500,001 (index 500,000). Time until loaded. Fields: `ms`.
3. `scroll`: 200 steps, every 16 ms set the top visible row to `500,000 + k*25`. Then wait until loaded. Fields: `total_ms`, `max_gap_ms`, `p95_gap_ms`, `placeholder_steps` (steps at which the visible range still had any `…` row when the next step began).
4. `cards_scroll`: switch to the Cards view, go to row 200,001 (index 200,000), wait until loaded, then 200 steps, every 16 ms set the top visible card to `200,000 + k*10`, then wait until loaded. Fields: `switch_ms` (switch plus go-to until loaded), `total_ms`, `max_gap_ms`, `p95_gap_ms`, `placeholder_steps`. Then switch back to the Table view.
5. `sort`: sort by Name descending. Time until loaded. Fields: `ms`.
6. `search`: set search to `steel` (no debounce in bench mode). Time until loaded and the count is updated. Fields: `ms`, `total`.
7. `export`: export the current query. Fields: `rows`, `ms`, `max_gap_ms`, `p95_gap_ms`.
8. `done`, then the desktop app quits (web: the page stays open).

Use `gguix/core/bench.Run(ctx, svc, driver)` (see `go doc gguix/core/bench`), which implements the steps, timing and metric output; your app implements the `bench.Driver` interface (UI-thread scheduling, switch view, jump, scroll-to-row, sort, search, start export, loaded check, quit). Use the same driver for the web target. <!--@fyne,cogent-->
Use `gguix/core/bench.Run(ctx, svc, driver)` (see `go doc gguix/core/bench`), which implements the steps, timing and metric output; your app implements the `bench.Driver` interface (UI-thread scheduling, switch view, jump, scroll-to-row, sort, search, start export, loaded check, quit). <!--@miqt-->
Implement the script in TypeScript under `frontend/src/bench/`, emitting each metric with `console.log("GGUIX_METRIC " + json)`. In the desktop app, also forward each line to stdout through a bound Go method `EmitMetric(event, json)` (which calls `core.EmitMetric`), and quit the app after `done`. `make bench` runs the desktop app with `GGUIX_BENCH=1`; the frontend learns about bench mode from a bound method. <!--@wails-->
Implement the script as a JavaScript file under `static/bench/`, included only when bench mode is on, emitting each metric with `console.log("GGUIX_METRIC " + json)`. In the desktop shell, also `POST` each line to `/bench/metric`, where the server writes it to stdout via `core.EmitMetric`, and quit after `done`. `make bench` runs the desktop shell with `GGUIX_BENCH=1`, which makes the server enable bench mode for the page. <!--@gotth-->

`make bench-web` serves the web target. Opening it in a browser with `?bench=1` must run the same script and print the same metric lines to the browser console. <!--@fyne,cogent,wails,gotth-->

---

## 10. Results files

### 10.1 `results/{{ID}}/toolchain.md`

A table of every tool and system dependency beyond `go` that building or running needed: name, exact version, install command, approximate install size, needed for (dev / build / runtime / web / cross), and whether it is cgo-related. Then a section `Clean machine steps`: the exact ordered commands from a Mac with only Go, Homebrew and the Xcode Command Line Tools installed to a running desktop app, and then to a running web app (if supported).

### 10.2 `results/{{ID}}/friction.md`

Chronological entries:

```
### <HH:MM> <short title>
- Step: <what you were doing>
- Command/API: <exact command or API>
- Outcome: <what happened>
- Time lost: <minutes>
- Category: docs-gap | api-confusion | toolchain | build-error | runtime-bug | framework-limitation | core-issue
- Resolution: <what fixed it, or "unresolved, recorded as gap">
```

End with a summary: wall time to first window, wall time to feature complete, count of entries per category, and the three biggest pain points.

### 10.3 `results/{{ID}}/capabilities.md`

One row per requirement from sections 5, 6, 7 and 8: `requirement | status (native / framework-drawn / fallback / gap / not feasible) | how it was done | notes`. It must include: each form control, split view, column resize, menus, shortcuts, file dialogs (native or not), confirm dialog, clipboard, notifications, theme modes and live system switching, UI-thread mechanism, the native prefs facility (unused), each custom cell and the card item (mechanism, files, and any visual compromise), the host panel per target (source, populated fields, reveal actions, mechanism), each web target, `cgoprobe` per target, cross builds, and each test scenario.

It ends with a section `Target notes`: a few sentences on what had to differ between the desktop and web targets and why, pointing at the marked files and blocks (section 3.3). <!--@fyne,cogent,wails,gotth-->

---

## 11. Definition of done

- All make targets exist and either work or print `UNSUPPORTED` with a reason.
- `make build`, `make run`, `make test` and `make bench` work on this Mac.
- `make bench` prints all eight `GGUIX_METRIC` events.
- Every section 5 behavior is implemented or recorded as a gap with a fallback.
- `toolchain.md`, `friction.md` and `capabilities.md` are complete.
- `README.md` lets a person reproduce the build from a clean machine.
- No files outside `apps/{{ID}}/` and `results/{{ID}}/` were changed (global tool installs excepted, all listed in `toolchain.md`).
- Your final reply summarizes what works, the gaps you recorded, and the `GGUIX_TESTS` line.

<!-- @audience master -->
---

## Appendix A. Metrics collected by the main model

All written to `results/<id>/metrics.json` by `bench/measure.sh <id>`, run sequentially. Desktop runs are repeated 5 times; medians are reported with min/max.

| Group | Metric | How |
|---|---|---|
| Bootstrap | Steps and wall time from clean to first window | `friction.md` and the agent's run time |
| Toolchain | Tools beyond `go`, versions, install sizes, npm direct/transitive package count, `npm audit` summary, whether plain `go build`/`go run .` works | `toolchain.md` plus `du`, `npm ls --all` |
| Build | Cold build time (fresh `GOCACHE` in a temp dir, warm module cache), `make deps` time | `bench/measure.sh` |
| Build | Incremental build after changing `BuildMarker`; for `wails`/`gotth` also after `UIMarker` | `sed` then time `make build` |
| Runtime | Binary size, `.app` size, external dylibs outside `/System` and `/usr/lib` (`otool -L`), after `macdeployqt` for `miqt` | `bench/measure.sh` |
| Runtime | Idle RSS 5 s after ready, peak RSS during the bench script | `ps` sampling every 100 ms |
| Perf | All section 9.2 events | `GGUIX_METRIC` lines |
| Web | Built web bundle size (raw and gzip), initial transfer size, web bench events, JS heap | `bench/web.mjs` (Playwright + Chrome, opens `http://localhost:8090/?bench=1`) |
| Reach | `cgoprobe` results per target, cross-compile results | make targets |
| Code | Lines of code by language excluding generated, vendored and `node_modules`; number of files; longest function | `bench/loc.sh` |
| Code | **Dual-target overhead**: lines per section 3.3 category, overhead lines (`desktop + web`) and overhead percentage of `shared + desktop + web`; number of build-tagged files and inline marker blocks. `miqt`: `n/a (no web target)` | `bench/loc.sh` |
| Code | Custom rendering size: lines in the files implementing the custom cells and the card item (listed in `capabilities.md`) | `bench/loc.sh` |
| Host | Host panel `Source` and populated fields per target, and whether reveal actions worked | `capabilities.md`, spot-checked by the main model |
| Tests | passed / failed / not feasible | `make test` |
| Style | Screenshots, light and dark, desktop and web, same state (row 1 selected, no filter), table and cards views | captured by the main model |

## Appendix B. Judge rubric

The judge scores each factor 1 to 5 per app with a two-to-four sentence justification citing files, metrics or log entries. No weighting and no overall winner; the report lists strengths and "best for" recommendations.

| Factor | 5 means | 1 means |
|---|---|---|
| Bootstrap and build loop | One documented command to a running app; fast incremental builds; hot reload | Many manual steps, workarounds, slow rebuilds |
| Runtime requirements | Single self-contained binary or bundle, no external libraries | Requires the user to install large libraries or runtimes |
| Toolchain footprint | `go` plus at most a Go-installed CLI | Several non-Go toolchains (Node, Qt, C++ SDKs), large installs |
| Native widgets and OS integration | All section 5 controls and OS features native, no gaps | Many gaps or drawn imitations |
| Desktop and web reach | Same app runs on desktop and web; the lowest dual-target overhead among the apps that have a web route earns the most credit | No web route |
| Core-constraint tolerance | Every target still works with `cgoprobe`, or only needs the shared remote mode | Web unusable with a cgo core and no workable split |
| Code clarity and complexity | Small, readable, idiomatic, clear separation of UI and logic; target markers used honestly | Large, tangled, lots of glue, unmarked target branches |
| Testability | All T1 to T12 automated with the framework's tools, simply | Few scenarios feasible, heavy harnesses |
| Performance | Fast ready/jump/sort, no placeholder steps, low tick gaps and memory, in both views | Slow, janky, high memory |
| Out-of-box style appeal | Blind screenshot rating: polished, modern, consistent in both themes | Dated or inconsistent |
| Native host access | Running user and host details available and correctly labeled in desktop and web-in-desktop mode, host actions work, little glue code | Identity unavailable or mislabeled, host actions missing |
| Custom rendering | Custom cells and cards match the spec closely with idiomatic, compact code and no scrolling jank | Visual compromises, heavy code, or slow scrolling |

## Appendix C. Briefs, workspaces and agent prompts

**Generation**: `cd tools && go run ./mkbrief -spec ../SPEC.md -out ../briefs` writes `briefs/<id>.md` for all five apps:

- drops `master` blocks and lines; keeps `brief` blocks; keeps ID-tagged blocks and lines only for matching IDs; strips all tag comments; substitutes `{{ID}}` and `{{NAME}}` (`Fyne`, `Cogent Core`, `Wails`, `MIQT`, `GoTTH`); collapses repeated blank lines and repeated `---` separators.

**Leak check** (the generator fails if any brief contains a term belonging to another app, case-sensitive, word-bounded):

| App | Terms that must not appear in other apps' briefs |
|---|---|
| `fyne` | `Fyne`, `fyne` |
| `cogent` | `Cogent`, `cogent`, `cogentcore` |
| `wails` | `Wails`, `wails`, `wails3`, `Svelte`, `svelte`, `shadcn-svelte`, `TanStack`, `tanstack`, `Vitest` |
| `miqt` | `MIQT`, `miqt`, `Qt`, `QTableView`, `QStyledItemDelegate` |
| `gotth` | `GoTTH`, `gotth`, `templ`, `htmx`, `shadcn-templ`, `templUI`, `webview_go` |

It also fails if any brief contains (case-insensitive, word-bounded): `judge`, `rubric`, `score`, `scoring`, `overhead`, `measured`, `compare`, `comparison`, `experiment`, `other implementations`, or `master`; or any leftover tag (`@audience`, `<!--@`, `{{`). Mentions of the bench and its metric lines are allowed: implementers need to build bench mode, and knowing that it reports timings does not reveal how anything is judged.

**Workspace setup** (main model, per app): create `~/src/gguix-work/<id>/`, copy `core/` into it, create empty `apps/<id>/` and `results/<id>/`, copy `briefs/<id>.md` to `BRIEF.md`.

**Agent prompt** (identical for all five except the path; model Sonnet 5.5; run in the background, all five in parallel):

```
Your workspace root is ~/src/gguix-work/<id>.
Read BRIEF.md in that directory and implement everything it specifies, following it exactly.
Work only inside the workspace root, apart from the global tool installs the brief calls for.
When you are done, reply with the summary described in the brief's Definition of done.
```
<!-- @end -->
