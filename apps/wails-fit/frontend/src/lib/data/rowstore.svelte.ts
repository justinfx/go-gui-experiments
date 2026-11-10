// RowStore: the lazily fetching row model behind the table and the card list.
//
// It plays the role Qt's QAbstractItemModel (canFetchMore / fetchMore) or a
// Svelte store plays in other toolkits, built on Svelte 5 reactivity:
//
//  * Rows live in fixed pages in a SvelteMap, so a view that calls row(i)
//    subscribes to exactly the page that holds i. A page arriving re-renders
//    only the rows of that page. There is no global "version" counter that
//    makes every visible row refresh on every response.
//  * Fetching is driven by where the view is going, not only where it is. The
//    store measures scroll velocity and reads ahead in the direction of travel,
//    so rows are usually there before they scroll into view.
//  * Every setVisible() recomputes the wanted set from scratch. Pages the user
//    has already scrolled past are never queued, and requests that are still
//    in flight are kept only if their page is wanted when they land.
//  * Wanted pages are sent together as one batch call (see PageBatch in
//    adapter_pages.go), which Go fans out in parallel, so a batch costs one
//    service latency whatever its size.
//  * The status-bar numbers (cached pages, last fetch) are throttled; the
//    row and total state is updated at once.
//
// Limits: at most pageSize (200) rows per service range, maxPages (40) pages
// cached (8,000 rows), and only pages near the viewport are ever requested.
import { SvelteMap } from "svelte/reactivity";

export interface RowStoreOptions {
  /** Rows per page; also the rows per service call. Never above 200. */
  pageSize: number;
  /** Cached pages; pageSize * maxPages must stay below 10,000 rows. */
  maxPages: number;
  /** Pages in one batch call. */
  maxBatchPages: number;
  /** Concurrent batch calls for read-ahead. Visible pages may use two more. */
  maxInFlight: number;
  /** Rows kept ready on the side away from the direction of travel (and both sides at rest). */
  marginRows: number;
  /** Largest read-ahead, in rows. */
  aheadMaxRows: number;
  /** How far ahead, in time, to read at the current scroll speed. */
  lookaheadMs: number;
  /** Delay before a failed request is retried. */
  retryMs: number;
  /** Minimum interval between updates of the stats object. */
  statsMs: number;
}

export const defaultRowStoreOptions: RowStoreOptions = {
  pageSize: 200,
  maxPages: 40,
  maxBatchPages: 4,
  maxInFlight: 3,
  marginRows: 50,
  aheadMaxRows: 800,
  lookaheadMs: 400,
  retryMs: 1000,
  statsMs: 100,
};

export interface RowStats {
  /** Milliseconds taken by the most recent completed batch call. */
  lastFetch: number;
  cachedPages: number;
  inFlight: number;
}

export interface FetchedPage<T> {
  offset: number;
  items: T[];
  total: number;
}
export interface RowRange {
  offset: number;
  limit: number;
}
export type BatchFetcher<T, Q> = (query: Q, ranges: RowRange[]) => Promise<FetchedPage<T>[]>;

/** A scroll that moves further than this between two reports is a jump, not a drag. */
const JUMP_ROWS = 5000;

export class RowStore<T, Q> {
  readonly opts: RowStoreOptions;
  private fetcher: BatchFetcher<T, Q>;
  private onChange: () => void;

  // Reactive state read by the views.
  private pages = new SvelteMap<number, T[]>();
  private _total = $state(-1);
  private _stats = $state<RowStats>({ lastFetch: 0, cachedPages: 0, inFlight: 0 });

  // Plain state.
  private gen = 0;
  private q: Q | undefined;
  private started = false;
  private stale = new Set<number>();
  private inFlight = new Set<number>();
  private calls = 0;
  private failed = new Map<number, number>();
  private pinned = new Set<number>();
  private waiters = new Map<number, (() => void)[]>();
  private first = 0;
  private last: number;
  private vel = 0; // rows per second, signed
  private lastReport = 0;
  private lastFetch = 0;
  private statsTimer: ReturnType<typeof setTimeout> | undefined;
  private statsAt = 0;
  private closed = false;

  constructor(fetcher: BatchFetcher<T, Q>, onChange: () => void, opts: Partial<RowStoreOptions> = {}) {
    this.opts = { ...defaultRowStoreOptions, ...opts };
    if (this.opts.pageSize > 200 || this.opts.pageSize * this.opts.maxPages > 10_000) throw new Error("RowStore limits exceeded");
    this.fetcher = fetcher;
    this.onChange = onChange;
    this.last = this.opts.pageSize - 1;
  }

  // ---- queries ----

  /** Switches to a new query: clears the cache, bumps the generation, moves
   * the visible range to the top (keeping its height) and fetches page 0. */
  setQuery(q: Q): void {
    this.bump();
    this.q = q;
    this.started = true;
    this._total = -1;
    this.pages.clear();
    this.stale.clear();
    this.pinned.clear();
    this.vel = 0;
    this.lastReport = 0;
    const height = Math.max(0, this.last - this.first);
    this.first = 0;
    this.last = height;
    this.resolveAllWaiters();
    this.dispatch();
    this.touchStats(true);
    this.onChange();
  }

  query(): Q | undefined {
    return this.q;
  }

  /** Refetches the visible pages of the current query (for example after an
   * update). They keep serving rows until fresh data arrives; all other
   * cached pages are dropped. */
  invalidate(): void {
    if (!this.started) return;
    this.bump();
    for (const k of [...this.pages.keys()]) {
      if (this.isVisible(k)) this.stale.add(k);
      else this.pages.delete(k);
    }
    this.dispatch();
    this.touchStats(true);
  }

  // ---- view interface ----

  /** The view reports the inclusive range of rows it shows. Schedules fetches. */
  setVisible(first: number, last: number): void {
    first = Math.max(0, first);
    if (last < first) last = first;
    const now = performance.now();
    if (first !== this.first && this.lastReport > 0) {
      const dt = now - this.lastReport;
      const dist = first - this.first;
      if (dt > 0 && dt < 400 && Math.abs(dist) < JUMP_ROWS) this.vel = 0.3 * this.vel + (0.7 * dist * 1000) / dt;
      else this.vel = 0;
    }
    if (first !== this.first) this.lastReport = now;
    this.first = first;
    this.last = last;
    this.evict();
    this.dispatch();
  }

  visibleRange(): [number, number] {
    return [this.first, this.last];
  }

  /** Rows in the current result, or -1 until the first page of the query arrives. Reactive. */
  total(): number {
    return this._total;
  }

  /** Row i, or undefined while its page is not loaded (show a placeholder).
   * Never blocks. Reactive per page. */
  row(i: number): T | undefined {
    if (i < 0) return undefined;
    return this.pages.get(Math.floor(i / this.opts.pageSize))?.[i % this.opts.pageSize];
  }

  /** True when the total is known and every row in [first, min(last, total-1)] is loaded. */
  rangeLoaded(first: number, last: number): boolean {
    if (this._total < 0) return false;
    last = Math.min(last, this._total - 1);
    for (let i = Math.max(0, first); i <= last; i++) {
      if (this.row(i) === undefined) return false;
    }
    return true;
  }

  /** Resolves with row i once it is loaded, asking for its page if needed.
   * Resolves undefined if the query changes first. */
  whenLoaded(i: number): Promise<T | undefined> {
    const have = this.row(i);
    if (have !== undefined) return Promise.resolve(have);
    const k = Math.floor(i / this.opts.pageSize);
    this.pinned.add(k);
    this.dispatch();
    return new Promise((resolve) => {
      const list = this.waiters.get(k) ?? [];
      list.push(() => resolve(this.row(i)));
      this.waiters.set(k, list);
    });
  }

  /** Finds a loaded row by predicate (cached pages only). Returns its index or -1. */
  findLoaded(pred: (item: T) => boolean): number {
    for (const [k, items] of this.pages) {
      for (let j = 0; j < items.length; j++) {
        if (pred(items[j])) return k * this.opts.pageSize + j;
      }
    }
    return -1;
  }

  /** Cache and fetch numbers for the status bar. Reactive, throttled. */
  stats(): RowStats {
    return this._stats;
  }

  close(): void {
    this.closed = true;
    this.started = false;
    this.bump();
    this.resolveAllWaiters();
  }

  // ---- fetch planning ----

  private bump(): void {
    this.gen++;
    this.inFlight.clear();
    this.failed.clear();
    this.calls = 0;
  }

  private isVisible(k: number): boolean {
    const ps = this.opts.pageSize;
    return k * ps <= this.last && (k + 1) * ps - 1 >= this.first;
  }

  /** Pages worth having right now, most urgent first. */
  private wanted(): number[] {
    const ps = this.opts.pageSize;
    const o = this.opts;
    const anchor = Math.floor(this.first / ps);
    if (this._total < 0) return [anchor];
    const speed = Math.abs(this.vel);
    const ahead = speed < 100 ? o.marginRows : Math.min(o.aheadMaxRows, Math.max(o.marginRows, (speed * o.lookaheadMs) / 1000));
    const fwd = this.vel >= 0;
    const lo = Math.max(0, this.first - (fwd ? o.marginRows : ahead));
    const hi = Math.min(this._total - 1, this.last + (fwd ? ahead : o.marginRows));
    if (hi < lo) return [];
    const head = fwd ? anchor : Math.floor(this.last / ps);
    const vis: number[] = [];
    const near: number[] = [];
    for (let k = Math.floor(lo / ps); k <= Math.floor(hi / ps); k++) (this.isVisible(k) ? vis : near).push(k);
    const byDist = (a: number, b: number) => Math.abs(a - head) - Math.abs(b - head) || a - b;
    vis.sort(byDist);
    near.sort(byDist);
    const pins = [...this.pinned].filter((k) => !vis.includes(k));
    return [...vis, ...pins, ...near];
  }

  private dispatch(): void {
    if (!this.started || this._total === 0) return;
    const now = Date.now();
    const need = this.wanted().filter((k) => {
      if (this.inFlight.has(k)) return false;
      if (this.pages.has(k) && !this.stale.has(k)) return false;
      const t = this.failed.get(k);
      return t === undefined || now - t >= this.opts.retryMs;
    });
    while (need.length > 0) {
      const urgent = this.isVisible(need[0]) || this.pinned.has(need[0]);
      if (this.calls >= this.opts.maxInFlight + (urgent ? 2 : 0)) break;
      this.send(need.splice(0, this.opts.maxBatchPages));
    }
  }

  private send(batch: number[]): void {
    const gen = this.gen;
    const ps = this.opts.pageSize;
    const start = performance.now();
    this.calls++;
    for (const k of batch) this.inFlight.add(k);
    this.touchStats();
    const ranges = batch.map((k) => ({ offset: k * ps, limit: ps }));
    this.fetcher(this.q as Q, ranges).then(
      (res) => {
        if (gen !== this.gen || this.closed) return;
        this.calls--;
        for (const k of batch) this.inFlight.delete(k);
        this.lastFetch = performance.now() - start;
        this.apply(res);
      },
      () => {
        if (gen !== this.gen || this.closed) return;
        this.calls--;
        const t = Date.now();
        for (const k of batch) {
          this.inFlight.delete(k);
          this.failed.set(k, t);
        }
        setTimeout(() => {
          if (gen === this.gen && !this.closed) this.dispatch();
        }, this.opts.retryMs);
        this.touchStats();
        this.dispatch();
      },
    );
  }

  private apply(res: FetchedPage<T>[]): void {
    const wasUnknown = this._total < 0;
    const ps = this.opts.pageSize;
    const keep = new Set(this.wanted());
    const fresh: number[] = [];
    for (const pg of res) {
      const k = Math.floor(pg.offset / ps);
      this.failed.delete(k);
      this._total = pg.total;
      // A page nobody wants any more is kept only if there is room.
      if (keep.has(k) || this.pages.size < this.opts.maxPages) {
        this.pages.set(k, pg.items);
        this.stale.delete(k);
        this.pinned.delete(k);
        fresh.push(k);
      }
    }
    const maxPage = this._total > 0 ? Math.floor((this._total - 1) / ps) : -1;
    for (const k of [...this.pages.keys()]) if (k > maxPage) this.pages.delete(k);
    this.evict();
    for (const k of fresh) {
      const list = this.waiters.get(k);
      if (list) {
        this.waiters.delete(k);
        for (const w of list) w();
      }
    }
    this.dispatch();
    this.touchStats(wasUnknown);
    this.onChange();
  }

  /** Keeps at most maxPages pages, dropping the ones furthest from the viewport. */
  private evict(): void {
    if (this.pages.size <= this.opts.maxPages) return;
    const ps = this.opts.pageSize;
    const mid = (this.first + this.last) / 2 / ps;
    const keys = [...this.pages.keys()].filter((k) => !this.isVisible(k)).sort((a, b) => Math.abs(b - mid) - Math.abs(a - mid));
    for (const k of keys) {
      if (this.pages.size <= this.opts.maxPages) break;
      this.pages.delete(k);
      this.stale.delete(k);
    }
  }

  private resolveAllWaiters(): void {
    const all = [...this.waiters.values()].flat();
    this.waiters.clear();
    for (const w of all) w();
  }

  private touchStats(now = false): void {
    const t = Date.now();
    const write = () => {
      this.statsTimer = undefined;
      this.statsAt = Date.now();
      this._stats = { lastFetch: this.lastFetch, cachedPages: this.pages.size, inFlight: this.inFlight.size };
    };
    if (now || t - this.statsAt >= this.opts.statsMs) {
      clearTimeout(this.statsTimer);
      write();
    } else if (this.statsTimer === undefined) {
      this.statsTimer = setTimeout(write, this.opts.statsMs - (t - this.statsAt));
    }
  }
}
