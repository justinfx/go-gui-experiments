// Application state: preferences, the query and its pager, the selection, the
// export job and the dialogs. Components read this singleton.
import { Events } from "@wailsio/runtime";
import { setMode } from "mode-watcher";
import { toast } from "svelte-sonner";
import { api, type ItemView, type QueryArg, type models } from "./api";
import { Pager } from "./pager";
import { palette } from "./palette.svelte";
import { FormModel } from "./form.svelte";

export type View = "table" | "cards";
export type Theme = "system" | "light" | "dark";

export interface ViewHandle {
  scrollToRow(index: number): void;
  ensureVisible(index: number): void;
  topRow(): number;
  visibleRange(): [number, number];
  focus(): void;
  element(): HTMLElement | null;
}

export interface ConfirmRequest {
  title: string;
  confirmLabel: string;
  cancelLabel: string;
  onConfirm: () => void;
}

class AppState {
  info = $state<models.AppInfo | null>(null);
  host = $state<models.HostView | null>(null);
  ready = $state(false);
  bench = $state(false);
  benchNoDebounce = false;

  theme = $state<Theme>("system");
  view = $state<View>("table");
  sortBy = $state("id");
  desc = $state(false);
  searchInput = $state("");
  search = $state("");
  categories = $state<string[]>([]);
  status = $state("");
  selectedId = $state(0);
  selectedIndex = $state(-1);

  /** Bumped whenever the pager's data or total changes. */
  version = $state(0);
  /** Bumped when a new query is applied (views scroll to the top). */
  queryGen = $state(0);
  totalText = $state("");
  visFirst = $state(0);
  visLast = $state(-1);
  scrollRequest = $state<{ index: number; seq: number } | null>(null);

  form = new FormModel();

  exportRunning = $state(false);
  exportDone = $state(0);
  exportTotal = $state(0);
  hasExport = $state(false);
  lastExportRows = 0;
  message = $state("");

  hostOpen = $state(false);
  aboutOpen = $state(false);
  confirm = $state<ConfirmRequest | null>(null);
  errorDialog = $state<{ title: string; message: string } | null>(null);

  active: ViewHandle | null = null;
  focusSearchFn: (() => void) | null = null;
  importFileFn: (() => void) | null = null;

  pager!: Pager<ItemView, QueryArg>;
  private seq = 0;
  private searchTimer: ReturnType<typeof setTimeout> | undefined;
  private messageTimer: ReturnType<typeof setTimeout> | undefined;
  private saveTimer: ReturnType<typeof setTimeout> | undefined;
  private pendingSelectionCheck = false;
  private abort: AbortController | null = null;
  private inited = false;

  get total(): number {
    void this.version;
    return this.pager ? this.pager.total() : -1;
  }
  get isDesktop(): boolean {
    return this.info?.target === "desktop";
  }
  get isWeb(): boolean {
    return this.info?.target === "web";
  }
  stats() {
    void this.version;
    return this.pager.stats();
  }
  row(i: number): ItemView | undefined {
    void this.version;
    return this.pager.row(i);
  }

  queryArg(): QueryArg {
    return { search: this.search, categories: [...this.categories], status: this.status, sortBy: this.sortBy, desc: this.desc };
  }

  // ---- startup ----

  async init(): Promise<void> {
    if (this.inited) return;
    this.inited = true;
    this.info = await api.AppInfo();
    this.bench = this.info.bench || (this.info.target === "web" && new URLSearchParams(location.search).get("bench") === "1");
    this.benchNoDebounce = this.bench;
    const [prefs, pal] = await Promise.all([api.LoadPrefs(), api.Palette()]);
    palette.set = pal;
    this.theme = (["system", "light", "dark"].includes(prefs.theme) ? prefs.theme : "system") as Theme;
    this.view = prefs.view === "cards" ? "cards" : "table";
    this.sortBy = prefs.sortBy || "id";
    this.desc = prefs.sortDesc;
    this.search = this.searchInput = prefs.search;
    this.categories = [...(prefs.categories ?? [])];
    this.status = prefs.status;
    this.selectedId = this.bench ? 0 : prefs.selectedId;
    if (this.bench) this.view = "table";
    setMode(this.theme);

    this.pager = new Pager<ItemView, QueryArg>(
      async (q, offset, limit) => {
        const p = await api.Page({ query: q, offset, limit });
        this.totalText = p.totalText;
        return { items: p.items ?? [], total: p.total };
      },
      () => this.onPagerChange(),
    );
    this.pager.setQuery(this.queryArg());
    this.refreshHost();
    this.registerEvents();
    this.hasExport = await api.HasLastExport();
    this.ready = true;
    if (!this.bench && this.selectedId) void this.restoreSelection();
  }

  private async restoreSelection(): Promise<void> {
    const id = this.selectedId;
    const row = await api.FindRow(this.queryArg(), id, 2000);
    if (row < 0) {
      this.selectedId = 0;
      this.selectedIndex = -1;
      return;
    }
    const item = await api.Get(id);
    this.selectedIndex = row;
    this.form.load(item);
    this.requestScroll(row);
  }

  async refreshHost(): Promise<void> {
    try {
      this.host = await api.HostInfo();
    } catch {
      /* the status bar shows "unknown user" */
    }
  }

  private registerEvents(): void {
    Events.On("export:progress", (e: { data: models.ExportProgress }) => {
      this.exportDone = e.data.done;
      this.exportTotal = e.data.total;
    });
    Events.On("export:done", (e: { data: models.ExportDone }) => {
      this.exportRunning = false;
      this.lastExportRows = e.data.rows;
      this.hasExport = this.isDesktop;
      const text = `${e.data.rows.toLocaleString("en-US")} rows written`;
      if (!e.data.notified) {
        toast.success("Export complete", { description: text });
        this.webNotify("Export complete", text);
      }
      this.flash("Export complete");
    });
    Events.On("export:error", (e: { data: models.ExportError }) => {
      this.exportRunning = false;
      this.errorDialog = { title: "Export failed", message: e.data.message };
    });
    Events.On("export:cancelled", () => {
      this.exportRunning = false;
      this.flash("Export cancelled");
    });
    Events.On("menu:action", (e: { data: models.MenuAction }) => void this.action(e.data.action));
    Events.On("app:flush", async () => {
      await this.savePrefsNow();
      await api.PrefsFlushed();
    });
  }

  private webNotify(title: string, body: string): void {
    try {
      if (this.isWeb && typeof Notification !== "undefined" && Notification.permission === "granted") new Notification(title, { body });
    } catch {
      /* in-app toast already shown */
    }
  }

  // ---- pager callbacks ----

  private onPagerChange(): void {
    this.version++;
    if (this.pendingSelectionCheck && this.pager.total() >= 0) {
      this.pendingSelectionCheck = false;
      if (this.selectedId) {
        const idx = this.pager.findLoaded((it) => it.id === this.selectedId);
        if (idx < 0 || idx >= this.pager.opts.pageSize) this.clearSelection();
        else this.selectedIndex = idx;
      }
    }
  }

  setVisible(first: number, last: number): void {
    this.visFirst = first;
    this.visLast = last;
    this.pager?.setVisible(first, last);
  }

  // ---- queries ----

  /** Applies the current search/filter/sort and reloads from the first page. */
  applyQuery(): void {
    this.pendingSelectionCheck = true;
    this.queryGen++;
    this.pager.setQuery(this.queryArg());
    this.schedulePrefsSave();
  }

  onSearchInput(text: string): void {
    this.searchInput = text;
    clearTimeout(this.searchTimer);
    if (this.benchNoDebounce) return;
    this.searchTimer = setTimeout(() => this.commitSearch(), 250);
  }

  commitSearch(): void {
    clearTimeout(this.searchTimer);
    if (this.search === this.searchInput) return;
    this.search = this.searchInput;
    this.applyQuery();
  }

  toggleCategory(name: string, on: boolean): void {
    const set = new Set(this.categories);
    if (on) set.add(name);
    else set.delete(name);
    this.categories = ["Electronics", "Garden", "Kitchen", "Office", "Outdoor", "Sports", "Tools", "Toys"].filter((c) => set.has(c));
    this.applyQuery();
  }

  clearCategories(): void {
    this.categories = [];
    this.applyQuery();
  }

  setStatus(status: string): void {
    this.status = status;
    this.applyQuery();
  }

  /** Header click: sort ascending, or toggle the direction of the active sort. */
  sortByField(field: string): void {
    if (this.sortBy === field) this.desc = !this.desc;
    else {
      this.sortBy = field;
      this.desc = false;
    }
    this.applyQuery();
  }

  setSort(field: string, desc: boolean): void {
    this.sortBy = field;
    this.desc = desc;
    this.applyQuery();
  }

  reload(): void {
    this.applyQuery();
  }

  // ---- view, theme ----

  setView(v: View): void {
    if (this.view === v) return;
    this.view = v;
    this.schedulePrefsSave();
  }

  setTheme(t: Theme): void {
    this.theme = t;
    setMode(t);
    this.schedulePrefsSave();
  }

  // ---- selection ----

  clearSelection(): void {
    this.selectedId = 0;
    this.selectedIndex = -1;
    this.form.load(null);
    this.schedulePrefsSave();
  }

  /** Selects an item, asking first if the form has unsaved edits. */
  requestSelect(item: ItemView, index: number): void {
    if (item.id === this.selectedId) {
      this.selectedIndex = index;
      return;
    }
    const go = () => {
      this.selectedId = item.id;
      this.selectedIndex = index;
      this.form.load(item);
      this.schedulePrefsSave();
    };
    if (this.form.dirty) this.askDiscard(go);
    else go();
  }

  selectIndex(index: number): void {
    const it = this.pager.row(index);
    if (it) {
      this.requestSelect(it, index);
      return;
    }
    void api.Page({ query: this.queryArg(), offset: index, limit: 1 }).then((p) => {
      if (p.items?.[0]) this.requestSelect(p.items[0], index);
    });
  }

  /** Up/Down arrow handling for both views. */
  moveSelection(delta: number): void {
    const total = this.pager.total();
    if (total <= 0) return;
    let cur = this.selectedId ? this.pager.findLoaded((it) => it.id === this.selectedId) : -1;
    if (cur < 0) cur = this.selectedIndex;
    const next = cur < 0 ? (delta > 0 ? 0 : total - 1) : Math.max(0, Math.min(total - 1, cur + delta));
    this.active?.ensureVisible(next);
    this.selectIndex(next);
  }

  askDiscard(onConfirm: () => void): void {
    this.confirm = { title: "Discard unsaved changes?", confirmLabel: "Discard", cancelLabel: "Cancel", onConfirm };
  }

  requestScroll(index: number): void {
    this.scrollRequest = { index, seq: ++this.seq };
  }

  /** Go-to-row: 1-based, clamped; scrolls so the row is on top and selects it. */
  goToRow(n: number): void {
    const total = this.pager.total();
    if (total <= 0 || !Number.isFinite(n)) return;
    const index = Math.max(1, Math.min(total, Math.trunc(n))) - 1;
    this.requestScroll(index);
    this.selectIndex(index);
  }

  /** The row index a freshly shown view should put on top. */
  initialTop(): number {
    if (this.selectedId) {
      const i = this.pager.findLoaded((it) => it.id === this.selectedId);
      if (i >= 0) return i;
      if (this.selectedIndex >= 0) return this.selectedIndex;
    }
    return 0;
  }

  // ---- saving ----

  async save(): Promise<void> {
    const f = this.form;
    if (!f.loaded || f.saving) return;
    f.saving = true;
    try {
      const res = await api.Update(f.toEdit());
      const errs = res.errors ?? [];
      if (errs.length > 0 || !res.item) {
        f.setErrors(errs);
        return;
      }
      f.setErrors([]);
      f.load(res.item);
      this.pager.invalidate();
      const row = await api.FindRow(this.queryArg(), res.item.id, 2000);
      if (row >= 0) this.selectedIndex = row;
    } catch (e) {
      this.errorDialog = { title: "Save failed", message: String(e) };
    } finally {
      f.saving = false;
    }
  }

  revert(): void {
    if (!this.form.dirty) return;
    this.askDiscard(() => this.form.reset());
  }

  // ---- clipboard, notes ----

  flash(text: string, ms = 2000): void {
    this.message = text;
    clearTimeout(this.messageTimer);
    this.messageTimer = setTimeout(() => (this.message = ""), ms);
  }

  async copyRow(): Promise<void> {
    const id = this.form.loaded?.id;
    if (!id) return;
    try {
      const r = await api.CopyRow(id);
      if (!r.copied) await navigator.clipboard.writeText(r.text);
      this.flash("Row copied");
    } catch (e) {
      this.errorDialog = { title: "Copy failed", message: String(e) };
    }
  }

  async importNotes(): Promise<void> {
    if (!this.form.loaded) {
      this.flash("Select an item first");
      return;
    }
    if (this.isWeb) {
      this.importFileFn?.();
      return;
    }
    try {
      const r = await api.ImportNotes();
      if (r.cancelled) return;
      if (r.error) {
        if (!r.errorShown) this.errorDialog = { title: "Import Notes", message: r.error };
        return;
      }
      this.form.notes = r.text;
    } catch (e) {
      this.errorDialog = { title: "Import Notes", message: String(e) };
    }
  }

  /** Browser file import (web target). */
  async importNotesFile(file: File): Promise<void> {
    if (file.size > 64 * 1024) {
      this.errorDialog = { title: "Import Notes", message: "File is too large." };
      return;
    }
    const text = new TextDecoder("utf-8").decode(await file.arrayBuffer());
    this.form.notes = Array.from(text).slice(0, 500).join("");
  }

  // ---- export ----

  async startExport(): Promise<void> {
    if (this.exportRunning) return;
    try {
      const res = await api.StartExport(this.queryArg());
      if (!res.started) return;
      this.exportRunning = true;
      this.exportDone = 0;
      this.exportTotal = 0;
      if (this.isWeb) await this.downloadStream(res.url);
    } catch (e) {
      this.exportRunning = false;
      this.errorDialog = { title: "Export failed", message: String(e) };
    }
  }

  /** Web target: read the streamed CSV. Outside bench mode it becomes a download. */
  private async downloadStream(url: string): Promise<void> {
    const ctl = new AbortController();
    this.abort = ctl;
    try {
      const r = await fetch(url, { signal: ctl.signal });
      if (!r.ok || !r.body) throw new Error(`Export request failed (${r.status})`);
      const reader = r.body.getReader();
      const chunks: Uint8Array[] = [];
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        if (!this.bench) chunks.push(value);
      }
      if (!this.bench) {
        const blob = new Blob(chunks as BlobPart[], { type: "text/csv" });
        const a = document.createElement("a");
        a.href = URL.createObjectURL(blob);
        a.download = "gguix-export.csv";
        document.body.appendChild(a);
        a.click();
        a.remove();
        setTimeout(() => URL.revokeObjectURL(a.href), 10_000);
      }
    } catch (e) {
      if ((e as Error).name === "AbortError") return;
      // The server reports failures with export:error as well.
      if (this.exportRunning) {
        this.exportRunning = false;
        this.errorDialog = { title: "Export failed", message: String(e) };
      }
    } finally {
      this.abort = null;
    }
  }

  async cancelExport(): Promise<void> {
    this.abort?.abort();
    await api.CancelExport();
  }

  // ---- host ----

  async openHost(): Promise<void> {
    await this.refreshHost();
    this.hostOpen = true;
  }

  async reveal(which: "config" | "export"): Promise<string | null> {
    try {
      if (which === "config") await api.RevealConfigFolder();
      else await api.RevealLastExport();
      return null;
    } catch (e) {
      return String((e as Error).message ?? e);
    }
  }

  // ---- actions (menu, shortcuts) ----

  async action(name: string): Promise<void> {
    switch (name) {
      case "about":
        this.aboutOpen = true;
        break;
      case "export":
        await this.startExport();
        break;
      case "import-notes":
        await this.importNotes();
        break;
      case "reveal-export": {
        const err = await this.reveal("export");
        if (err) this.errorDialog = { title: "Reveal Last Export", message: err };
        break;
      }
      case "copy-row":
        await this.copyRow();
        break;
      case "view-table":
        this.setView("table");
        break;
      case "view-cards":
        this.setView("cards");
        break;
      case "theme-system":
        this.setTheme("system");
        break;
      case "theme-light":
        this.setTheme("light");
        break;
      case "theme-dark":
        this.setTheme("dark");
        break;
      case "reload":
        this.reload();
        break;
      case "focus-search":
        this.focusSearchFn?.();
        break;
      case "host-info":
        await this.openHost();
        break;
    }
  }

  // ---- preferences ----

  prefsView(): models.PrefsView {
    return {
      theme: this.theme,
      sortBy: this.sortBy,
      sortDesc: this.desc,
      search: this.search,
      categories: [...this.categories],
      status: this.status,
      selectedId: this.selectedId,
      view: this.view,
      windowWidth: 0,
      windowHeight: 0,
    };
  }

  schedulePrefsSave(): void {
    if (!this.inited) return;
    clearTimeout(this.saveTimer);
    this.saveTimer = setTimeout(() => void this.savePrefsNow(), 500);
  }

  async savePrefsNow(): Promise<void> {
    clearTimeout(this.saveTimer);
    try {
      await api.SavePrefs(this.prefsView());
    } catch {
      /* preferences are best effort */
    }
  }

  /** For tests: forget all state. */
  _resetForTests(): void {
    this.inited = false;
  }
}

export const app = new AppState();
