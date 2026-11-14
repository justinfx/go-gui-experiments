// Pager: a TypeScript port of the algorithm in BRIEF section 4.6.1. It caches
// pages of a query result for a virtualized view and never blocks: row(i)
// returns undefined for rows that are not loaded.
//
//  1. Page k covers rows [k*pageSize, (k+1)*pageSize).
//  2. setVisible(first, last) needs the pages covering
//     [first-prefetchMargin, last+prefetchMargin] clamped to [0, total-1].
//     Pages that intersect [first, last] are "visible", the others "prefetch".
//     While total is unknown only the page containing first is requested.
//  3. Visible pages are fetched before prefetch pages, each group in
//     ascending distance from the page containing first, at most maxInFlight
//     requests at once. Cached or in-flight pages are not requested again.
//  4. Responses carry the generation they were requested under; older ones
//     are discarded.
//  5. The cache is LRU by last setVisible touch; over maxPages the least
//     recently touched pages that are not visible are evicted.
//  6. A failed request is retried after one second.
//  7. invalidate() bumps the generation, marks visible pages stale (they keep
//     serving row() until refetched) and drops all other pages.

export interface PagerOptions {
  pageSize: number;
  maxPages: number;
  maxInFlight: number;
  prefetchMargin: number;
  /** Retry delay after a failed request. The brief says one second. */
  retryMs: number;
}

export const defaultPagerOptions: PagerOptions = {
  pageSize: 200,
  maxPages: 20,
  maxInFlight: 2,
  prefetchMargin: 50,
  retryMs: 1000,
};

export interface PagerStats {
  /** Milliseconds taken by the most recent completed page request. */
  lastFetch: number;
  cachedPages: number;
  inFlight: number;
}

export interface FetchedPage<T> {
  items: T[];
  total: number;
}

export type PageFetcher<T, Q> = (query: Q, offset: number, limit: number) => Promise<FetchedPage<T>>;

interface CachedPage<T> {
  items: T[];
  touched: number;
  stale: boolean;
}

export class Pager<T, Q> {
  readonly opts: PagerOptions;
  private fetcher: PageFetcher<T, Q>;
  private onChange: () => void;

  private gen = 0;
  private q: Q | undefined;
  private started = false;
  private _total = -1;
  private pages = new Map<number, CachedPage<T>>();
  private inFlight = new Set<number>();
  private failed = new Map<number, number>();
  private first = 0;
  private last: number;
  private clock = 0;
  private _lastFetch = 0;
  private closed = false;

  constructor(fetcher: PageFetcher<T, Q>, onChange: () => void, opts: Partial<PagerOptions> = {}) {
    this.opts = { ...defaultPagerOptions, ...opts };
    this.fetcher = fetcher;
    this.onChange = onChange;
    this.last = this.opts.pageSize - 1;
  }

  /** Switches to a new query: clears the cache, bumps the generation, moves
   * the visible range to the top (keeping its height) and fetches page 0. */
  setQuery(q: Q): void {
    this.reset();
    this.q = q;
    this.started = true;
    this._total = -1;
    this.pages.clear();
    const height = Math.max(0, this.last - this.first);
    this.first = 0;
    this.last = height;
    this.schedule();
    this.notify();
  }

  query(): Q | undefined {
    return this.q;
  }

  /** Reports the inclusive range of rows the view shows and schedules the
   * fetches it needs. Does not call onChange. */
  setVisible(first: number, last: number): void {
    first = Math.max(0, first);
    if (last < first) last = first;
    this.first = first;
    this.last = last;
    this.clock++;
    const ps = this.opts.pageSize;
    for (let k = Math.floor(first / ps); k <= Math.floor(last / ps); k++) {
      const pg = this.pages.get(k);
      if (pg) pg.touched = this.clock;
    }
    this.schedule();
    this.evict();
  }

  visibleRange(): [number, number] {
    return [this.first, this.last];
  }

  /** Number of rows in the current result, or -1 until the first page of the
   * current query arrives. */
  total(): number {
    return this._total;
  }

  /** Row i if its page is loaded; undefined otherwise. Never blocks. */
  row(i: number): T | undefined {
    if (i < 0 || (this._total >= 0 && i >= this._total)) return undefined;
    const pg = this.pages.get(Math.floor(i / this.opts.pageSize));
    if (!pg) return undefined;
    return pg.items[i % this.opts.pageSize];
  }

  /** True when total is known and every row in [first, min(last, total-1)] is
   * loaded. An empty result counts as loaded. */
  rangeLoaded(first: number, last: number): boolean {
    if (this._total < 0) return false;
    last = Math.min(last, this._total - 1);
    for (let i = Math.max(0, first); i <= last; i++) {
      if (this.row(i) === undefined) return false;
    }
    return true;
  }

  /** Finds a loaded row by predicate (cached pages only). Returns its index or -1. */
  findLoaded(pred: (item: T) => boolean): number {
    for (const [k, pg] of this.pages) {
      for (let j = 0; j < pg.items.length; j++) {
        if (pred(pg.items[j])) return k * this.opts.pageSize + j;
      }
    }
    return -1;
  }

  /** Refetches the visible pages of the current query (for example after an
   * update), keeping their data available until fresh data arrives, and drops
   * all other cached pages. */
  invalidate(): void {
    if (!this.started) return;
    this.reset();
    for (const [k, pg] of [...this.pages]) {
      if (this.isVisible(k)) pg.stale = true;
      else this.pages.delete(k);
    }
    this.schedule();
  }

  stats(): PagerStats {
    return { lastFetch: this._lastFetch, cachedPages: this.pages.size, inFlight: this.inFlight.size };
  }

  /** Stops the pager. It must not be used afterwards. */
  close(): void {
    this.closed = true;
    this.started = false;
    this.gen++;
  }

  private reset(): void {
    this.gen++;
    this.inFlight.clear();
    this.failed.clear();
  }

  private isVisible(k: number): boolean {
    const start = k * this.opts.pageSize;
    const end = (k + 1) * this.opts.pageSize - 1;
    return start <= this.last && end >= this.first;
  }

  private schedule(): void {
    if (!this.started || this._total === 0) return;
    const ps = this.opts.pageSize;
    const anchor = Math.floor(this.first / ps);
    let order: number[];
    if (this._total < 0) {
      order = [anchor];
    } else {
      const lo = Math.max(0, this.first - this.opts.prefetchMargin);
      const hi = Math.min(this._total - 1, this.last + this.opts.prefetchMargin);
      if (hi < lo) return;
      const vis: number[] = [];
      const pre: number[] = [];
      for (let k = Math.floor(lo / ps); k <= Math.floor(hi / ps); k++) {
        (this.isVisible(k) ? vis : pre).push(k);
      }
      const byDist = (a: number, b: number) => Math.abs(a - anchor) - Math.abs(b - anchor) || a - b;
      vis.sort(byDist);
      pre.sort(byDist);
      order = [...vis, ...pre];
    }
    const now = Date.now();
    for (const k of order) {
      if (this.inFlight.size >= this.opts.maxInFlight) return;
      const pg = this.pages.get(k);
      if (pg && !pg.stale) continue;
      if (this.inFlight.has(k)) continue;
      const t = this.failed.get(k);
      if (t !== undefined && now - t < this.opts.retryMs) continue;
      this.inFlight.add(k);
      void this.fetchPage(this.gen, this.q as Q, k);
    }
  }

  private async fetchPage(gen: number, q: Q, k: number): Promise<void> {
    const start = performance.now();
    let res: FetchedPage<T>;
    try {
      res = await this.fetcher(q, k * this.opts.pageSize, this.opts.pageSize);
    } catch {
      if (gen !== this.gen || this.closed) return;
      this.inFlight.delete(k);
      this.failed.set(k, Date.now());
      setTimeout(() => {
        if (gen === this.gen && !this.closed) this.schedule();
      }, this.opts.retryMs);
      this.schedule();
      return;
    }
    if (gen !== this.gen || this.closed) return;
    this.inFlight.delete(k);
    this.failed.delete(k);
    this.clock++;
    this.pages.set(k, { items: res.items, touched: this.clock, stale: false });
    this._total = res.total;
    this._lastFetch = performance.now() - start;
    const maxPage = Math.floor((this._total - 1) / this.opts.pageSize);
    for (const pk of [...this.pages.keys()]) {
      if (pk > maxPage) this.pages.delete(pk);
    }
    this.evict();
    this.schedule();
    this.notify();
  }

  private evict(): void {
    while (this.pages.size > this.opts.maxPages) {
      let victim = -1;
      let oldest = Infinity;
      for (const [k, pg] of this.pages) {
        if (this.isVisible(k)) continue;
        if (victim < 0 || pg.touched < oldest) {
          victim = k;
          oldest = pg.touched;
        }
      }
      if (victim < 0) return;
      this.pages.delete(victim);
    }
  }

  private notify(): void {
    this.onChange();
  }
}
