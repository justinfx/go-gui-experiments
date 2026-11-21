package ui

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/doug/gophics/shell"

	"gguix/core"
)

// ---- query ----

// SetSearchText is the search field's OnChange: it applies after 250 ms.
func (s *App) SetSearchText(v string) {
	s.Search = v
	if s.searchTimer != nil {
		s.searchTimer.Stop()
	}
	s.searchTimer = time.AfterFunc(250*time.Millisecond, func() {
		s.post(func() { s.SetState(func() { s.ApplySearch(s.Search) }) })
	})
}

// ApplySearch applies search text immediately (no debounce).
func (s *App) ApplySearch(v string) {
	s.Search = v
	q := s.Query
	q.Search = strings.TrimSpace(v)
	s.setQuery(q)
}

// ToggleCategory flips one category in the filter.
func (s *App) ToggleCategory(c core.Category) {
	q := s.Query
	cats := slices.Clone(q.Categories)
	if i := slices.Index(cats, c); i >= 0 {
		cats = slices.Delete(cats, i, i+1)
	} else {
		cats = append(cats, c)
	}
	slices.Sort(cats)
	q.Categories = cats
	s.setQuery(q)
}

// ClearCategories removes the category filter.
func (s *App) ClearCategories() {
	q := s.Query
	q.Categories = nil
	s.setQuery(q)
}

// SetStatusFilter sets the status filter; nil = all.
func (s *App) SetStatusFilter(st *core.Status) {
	q := s.Query
	q.Status = st
	s.setQuery(q)
}

// HeaderClick sorts by field ascending, or toggles direction when active.
func (s *App) HeaderClick(f core.SortField) {
	cur := s.Query.SortBy
	if cur == "" {
		cur = core.SortID
	}
	if cur == f {
		s.SortBy(f, !s.Query.Desc)
	} else {
		s.SortBy(f, false)
	}
}

// SortBy applies a sort.
func (s *App) SortBy(f core.SortField, desc bool) {
	q := s.Query
	q.SortBy, q.Desc = f, desc
	s.setQuery(q)
}

// Reload reloads the current query from the first page.
func (s *App) Reload() { s.setQuery(s.Query) }

func (s *App) setQuery(q core.Query) {
	s.Query = q
	s.Pager.SetQuery(q)
	s.checkSelReload = s.SelectedID != 0
	s.pendingSelectRow = -1
	s.scrollRowToTop(0)
	s.schedulePrefsSave()
}

// ---- scrolling and visible range ----

func (s *App) extent() float32 {
	if s.View == core.ViewCards {
		return CardExtent
	}
	return TableRowH
}

func (s *App) ctl() interface {
	JumpTo(float32)
	Offset() float32
	MaxOffset() float32
} {
	if s.View == core.ViewCards {
		return s.cardsCtl
	}
	return s.tableCtl
}

// visibleCount is the number of rows that fit the viewport.
func (s *App) visibleCount() int {
	h := s.viewportH
	if h <= 0 {
		h = 600
	}
	n := int((h + s.extent() - 1) / s.extent())
	return max(n, 1)
}

// scrollRowToTop makes row the top visible row and tells the pager.
func (s *App) scrollRowToTop(row int) {
	row = max(row, 0)
	s.ctl().JumpTo(float32(row) * s.extent())
	s.setVisibleFrom(row)
}

func (s *App) setVisibleFrom(first int) {
	total := s.Pager.Total()
	last := first + s.visibleCount() - 1
	if total >= 0 {
		last = min(last, total-1)
		first = min(first, max(total-1, 0))
	}
	s.First, s.Last = first, last
	s.Pager.SetVisible(first, max(last, first))
}

// syncVisible derives the visible range from the live scroll offset. It is
// called from the list's item builder, which LazyList runs whenever its
// offset changes.
func (s *App) syncVisible() {
	off := s.ctl().Offset()
	first := int(off / s.extent())
	if first != s.First || s.Last < first {
		s.setVisibleFrom(first)
		s.post(func() { s.SetState(func() {}) }) // refresh the status bar range
	}
}

// VisibleLoaded reports whether the current visible range has data.
func (s *App) VisibleLoaded() bool {
	total := s.Pager.Total()
	if total < 0 {
		return false
	}
	if total == 0 {
		return true
	}
	return s.Pager.RangeLoaded(s.First, max(s.Last, s.First))
}

// GoToRow makes the 0-based row the top visible row and selects it.
func (s *App) GoToRow(idx int) {
	total := s.Pager.Total()
	if total <= 0 {
		return
	}
	idx = min(max(idx, 0), total-1)
	s.scrollRowToTop(idx)
	if it, ok := s.Pager.Row(idx); ok {
		s.SelectItemGuarded(it)
	} else {
		s.guardDirty(func() { s.pendingSelectRow = idx })
	}
}

// GoToText parses the go-to field (1-based) and jumps.
func (s *App) GoToSubmit() {
	n, err := strconv.Atoi(strings.TrimSpace(strings.ReplaceAll(s.GoToText, ",", "")))
	if err != nil {
		return
	}
	s.GoToRow(n - 1)
}

// SetView switches between table and cards, keeping the selection visible.
func (s *App) SetView(v string) {
	if v != core.ViewCards {
		v = core.ViewTable
	}
	if v == s.View {
		return
	}
	s.View = v
	row := 0
	if s.SelectedID != 0 {
		if r, ok := s.rowOfSelected(); ok {
			row = r
		}
	}
	s.pendingTopRow = row
	s.setVisibleFrom(row)
	s.schedulePrefsSave()
}

func (s *App) rowOfSelected() (int, bool) {
	// The selected row is usually cached around the visible range.
	lo, hi := max(s.First-2000, 0), s.First+2000
	if t := s.Pager.Total(); t >= 0 {
		hi = min(hi, t-1)
	}
	for i := lo; i <= hi; i++ {
		if it, ok := s.Pager.Row(i); ok && it.ID == s.SelectedID {
			return i, true
		}
	}
	return 0, false
}

// SetTheme applies "system", "light" or "dark".
func (s *App) SetTheme(t string) {
	s.Theme = t
	s.schedulePrefsSave()
}

// restoreSelection finds the saved item within the first 2000 rows.
func (s *App) restoreSelection(id int64) {
	q := s.Query
	go func() {
		row, ok, err := core.FindRow(context.Background(), s.svc, q, id, 2000)
		if err != nil || !ok {
			return
		}
		s.Post(func() {
			s.pendingTopRow = row
			s.pendingSelectRow = row
			s.setVisibleFrom(row)
		})
	}()
}

// ---- selection ----

// SelectRow selects the item at row (guarded by the dirty check).
func (s *App) SelectRow(row int) {
	it, ok := s.Pager.Row(row)
	if !ok {
		return
	}
	s.SelectItemGuarded(it)
}

// SelectItemGuarded selects it, asking first if the form is dirty.
func (s *App) SelectItemGuarded(it core.Item) {
	if s.HasItem && it.ID == s.SelectedID {
		return
	}
	s.guardDirty(func() { s.selectItem(it) })
}

func (s *App) guardDirty(then func()) {
	if !s.Dirty() {
		then()
		return
	}
	s.confirmDiscard(then)
}

func (s *App) selectItem(it core.Item) {
	s.SelectedID = it.ID
	s.Item, s.HasItem = it, true
	s.Form = formFrom(it)
	s.Errors = map[string]string{}
	s.schedulePrefsSave()
}

func (s *App) clearSelection() {
	s.SelectedID = 0
	s.HasItem = false
	s.Item = core.Item{}
	s.Form = Form{}
	s.Errors = map[string]string{}
	s.schedulePrefsSave()
}

// MoveSelection moves the selection by delta rows (Up/Down keys).
func (s *App) MoveSelection(delta int) {
	total := s.Pager.Total()
	if total <= 0 {
		return
	}
	cur := -1
	if s.SelectedID != 0 {
		if r, ok := s.rowOfSelected(); ok {
			cur = r
		}
	}
	next := min(max(cur+delta, 0), total-1)
	if cur < 0 {
		next = s.First
	}
	it, ok := s.Pager.Row(next)
	if !ok {
		return
	}
	s.SelectItemGuarded(it)
	// Keep the selection in view.
	if next < s.First {
		s.scrollRowToTop(next)
	} else if next > s.First+s.visibleCount()-2 {
		s.scrollRowToTop(next - s.visibleCount() + 2)
	}
}

// RowSelected reports whether row i holds the selected item.
func (s *App) RowSelected(i int) bool {
	if s.SelectedID == 0 {
		return false
	}
	it, ok := s.Pager.Row(i)
	return ok && it.ID == s.SelectedID
}

// ---- form ----

// Dirty reports whether the form differs from the loaded item.
func (s *App) Dirty() bool {
	return s.HasItem && s.Form != formFrom(s.Item)
}

// EditForm applies a change to the draft.
func (s *App) EditForm(fn func(f *Form)) {
	if !s.HasItem {
		return
	}
	fn(&s.Form)
}

// parseForm builds the item from the draft, collecting parse errors.
func (s *App) parseForm() (core.Item, map[string]string) {
	errs := map[string]string{}
	it := s.Item
	f := s.Form
	it.Name = f.Name
	it.Category = core.Category(f.Category)
	if c, err := core.ParsePrice(f.Price); err != nil {
		errs["priceCents"] = core.ErrPriceFormat.Error()
	} else {
		it.PriceCents = c
	}
	if q, err := strconv.Atoi(strings.TrimSpace(strings.ReplaceAll(f.Qty, ",", ""))); err != nil {
		errs["quantity"] = core.MsgQuantity
	} else if q < 0 || q > core.MaxQuantity {
		errs["quantity"] = core.MsgQuantity
	} else {
		it.Quantity = int32(q)
	}
	it.Status = core.Status(f.Status)
	it.Rating = core.RatingFromTenths(f.Rating)
	it.Featured = f.Featured
	if strings.TrimSpace(f.Restock) == "" {
		it.RestockDate = nil
	} else if d, err := core.ParseDate(strings.TrimSpace(f.Restock)); err != nil {
		errs["restockDate"] = "Enter a date like 2026-03-07."
	} else {
		it.RestockDate = &d
	}
	it.Notes = f.Notes
	return it, errs
}

// Save validates the draft and calls DataService.Update.
func (s *App) Save() {
	if !s.HasItem || !s.Dirty() || s.saving {
		return
	}
	it, errs := s.parseForm()
	if ve := core.ValidateItem(it); ve != nil {
		for _, fe := range ve.Errors {
			if _, ok := errs[fe.Field]; !ok {
				errs[fe.Field] = fe.Message
			}
		}
	}
	s.Errors = errs
	if len(errs) > 0 {
		return
	}
	s.saving = true
	s.UpdateCalls++
	go func() {
		got, err := s.svc.Update(context.Background(), it)
		s.Post(func() {
			s.saving = false
			var ve *core.ValidationError
			switch {
			case errors.As(err, &ve):
				s.Errors = map[string]string{}
				for _, fe := range ve.Errors {
					s.Errors[fe.Field] = fe.Message
				}
			case err != nil:
				s.ShowMessage("Save failed", err.Error())
			default:
				s.Errors = map[string]string{}
				if got.ID == s.SelectedID {
					s.Item, s.Form = got, formFrom(got)
				}
				s.Pager.Invalidate()
			}
		})
	}()
}

// Revert asks, then reloads the form from the item.
func (s *App) Revert() {
	if !s.Dirty() {
		return
	}
	s.confirmDiscard(func() {
		s.Form = formFrom(s.Item)
		s.Errors = map[string]string{}
	})
}

// CopyRow copies the saved item's CSV row to the clipboard.
func (s *App) CopyRow() {
	if !s.HasItem {
		return
	}
	if cb := s.ctx.Clipboard(); cb != nil {
		if err := cb.ClipboardWrite(core.ItemCSVRow(s.Item)); err == nil {
			s.Flash("Row copied")
			return
		}
	}
	s.Flash("Clipboard unavailable")
}

// ImportNotes opens the file dialog and loads a .txt into Notes.
func (s *App) ImportNotes() {
	if !s.HasItem {
		return
	}
	fp := s.ctx.FilePicker()
	if fp == nil {
		s.ShowMessage("Import Notes", "File dialogs are not available on this platform.")
		return
	}
	fp.Open(shell.OpenOptions{Accept: []string{".txt", "text/plain"}}, func(files []shell.PickedFile, err error) {
		s.Post(func() {
			if err != nil {
				s.ShowMessage("Import failed", err.Error())
				return
			}
			if len(files) == 0 {
				return
			}
			s.ImportNotesData(files[0].Data)
		})
	})
}

// ImportNotesData applies imported file bytes to the Notes field.
func (s *App) ImportNotesData(data []byte) {
	if len(data) > core.MaxImportFileSize {
		s.ShowMessage("Import Notes", "File is too large.")
		return
	}
	txt := strings.ToValidUTF8(string(data), "�")
	if utf8.RuneCountInString(txt) > core.MaxNotesRunes {
		txt = string([]rune(txt)[:core.MaxNotesRunes])
	}
	s.EditForm(func(f *Form) { f.Notes = txt })
}

// ---- keyboard shortcuts ----

func (s *App) onKey(k shell.Key) {
	if k.Kind != shell.KeyPress {
		return
	}
	s.SetState(func() { s.handleKey(k) })
}

func (s *App) handleKey(k shell.Key) {
	if k.Mods.Command() {
		s.runShortcut(k)
		return
	}
	switch k.Code {
	case shell.KeyUp:
		s.MoveSelection(-1)
	case shell.KeyDown:
		s.MoveSelection(1)
	}
}

// isShortcut reports whether k is one of the app's command shortcuts. Plain
// Cmd+C/X/V/A/Z are not, so a focused field keeps its own editing keys.
func isShortcut(k shell.Key) bool {
	if k.Kind != shell.KeyPress || !k.Mods.Command() {
		return false
	}
	switch k.Code {
	case shell.KeyE, shell.KeyO, shell.Key1, shell.Key2, shell.KeyR, shell.KeyI, shell.KeyQ, shell.KeyF:
		return true
	case shell.KeyC:
		return k.Mods&shell.ModShift != 0
	}
	return false
}

// fieldKeyPreview is every text field's OnKeyPreview: keys reach only the
// focused widget (no bubbling), so a field must hand app shortcuts back.
func (s *App) fieldKeyPreview(k shell.Key) bool {
	if !isShortcut(k) {
		return false
	}
	s.SetState(func() { s.runShortcut(k) })
	return true
}

func (s *App) runShortcut(k shell.Key) {
	if !isShortcut(k) {
		return
	}
	switch k.Code {
	case shell.KeyE:
		s.StartExport()
	case shell.KeyO:
		s.ImportNotes()
	case shell.KeyC:
		s.CopyRow()
	case shell.Key1:
		s.SetView(core.ViewTable)
	case shell.Key2:
		s.SetView(core.ViewCards)
	case shell.KeyR:
		s.Reload()
	case shell.KeyI:
		s.ShowHost()
	case shell.KeyQ:
		s.Quit()
	case shell.KeyF:
		s.FocusSearch()
	}
}
