import { describe, expect, it, vi } from "vitest";
import { RowStore, type RowRange } from "./rowstore.svelte";

type Item = { id: number };
function make(total = 100_000, delay = 0, opts = {}) {
  const calls: RowRange[][] = [];
  let gate: (() => void)[] = [];
  let fail = false;
  const fetcher = vi.fn(async (_q: string, ranges: RowRange[]) => {
    calls.push(ranges);
    if (fail) throw new Error("boom");
    if (delay < 0) await new Promise<void>((r) => gate.push(r));
    else await new Promise((r) => setTimeout(r, delay));
    return ranges.map((r) => ({
      offset: r.offset,
      total,
      items: Array.from({ length: Math.max(0, Math.min(r.limit, total - r.offset)) }, (_, i) => ({ id: r.offset + i + 1 })),
    }));
  });
  const changes = { n: 0 };
  const s = new RowStore<Item, string>(fetcher, () => changes.n++, { retryMs: 20, statsMs: 0, ...opts });
  return { s, calls, changes, release: () => gate.splice(0).forEach((g) => g()), failNext: (f: boolean) => (fail = f) };
}
const wait = (ms = 15) => new Promise((r) => setTimeout(r, ms));
const offsets = (calls: RowRange[][]) => calls.flat().map((r) => r.offset);

describe("RowStore", () => {
  it("requests only the page of the first row while the total is unknown", async () => {
    const { s, calls } = make();
    s.setQuery("a");
    s.setVisible(0, 30);
    expect(offsets(calls)).toEqual([0]);
    expect(s.total()).toBe(-1);
    await wait();
    expect(s.total()).toBe(100_000);
    expect(s.row(0)?.id).toBe(1);
    expect(s.row(199)?.id).toBe(200);
  });

  it("sends the missing pages around the viewport as one batch with 200-row ranges", async () => {
    const { s, calls } = make();
    s.setQuery("a");
    await wait();
    calls.length = 0;
    s.setVisible(1000, 1190);
    expect(calls.length).toBe(1);
    expect(calls[0].every((r) => r.limit <= 200)).toBe(true);
    expect(offsets(calls)).toEqual([1000, 800, 1200]); // the visible page first, then its neighbours
  });

  it("reads ahead in the direction of travel and never requests skipped pages", async () => {
    const { s, calls } = make();
    s.setQuery("a");
    await wait();
    calls.length = 0;
    const t0 = performance.now();
    // 25 rows per step at about 16 ms: a fast drag.
    let first = 50_000;
    s.setVisible(first, first + 25);
    for (let k = 0; k < 12; k++) {
      await wait(16);
      first += 25;
      s.setVisible(first, first + 25);
    }
    await wait(80);
    void t0;
    const asked = offsets(calls);
    expect(asked.length).toBeGreaterThan(0);
    expect(Math.min(...asked)).toBeGreaterThanOrEqual(49_800);
    expect(Math.max(...asked)).toBeGreaterThan(first + 200); // read ahead
    expect(asked.every((o) => o % 200 === 0)).toBe(true);
    expect(s.rangeLoaded(first, first + 25)).toBe(true);
  });

  it("holds at most maxPages pages and 10,000 rows", async () => {
    const { s } = make(1_000_000);
    s.setQuery("a");
    await wait();
    for (let pos = 0; pos < 200_000; pos += 7_000) {
      s.setVisible(pos, pos + 30);
      await wait(2);
    }
    await wait(20);
    expect(s.stats().cachedPages).toBeLessThanOrEqual(40);
    expect(s.stats().cachedPages * s.opts.pageSize).toBeLessThanOrEqual(10_000);
  });

  it("discards responses of an older query", async () => {
    const h = make(1000, -1);
    h.s.setQuery("a");
    h.s.setQuery("b");
    h.release();
    await wait();
    expect(h.s.total()).toBe(1000);
    expect(h.s.stats().inFlight).toBeLessThanOrEqual(1);
  });

  it("keeps serving visible rows while invalidate refetches them", async () => {
    const h = make(1000, -1);
    h.s.setQuery("a");
    h.release();
    await wait();
    h.s.setVisible(0, 20);
    h.release();
    await wait();
    expect(h.s.row(5)?.id).toBe(6);
    h.s.invalidate();
    expect(h.s.row(5)?.id).toBe(6); // stale data stays until the new page lands
    h.release();
    await wait();
    expect(h.s.row(5)?.id).toBe(6);
  });

  it("whenLoaded waits for a far row without an extra service call", async () => {
    const { s, calls } = make();
    s.setQuery("a");
    await wait();
    const p = s.whenLoaded(7_000);
    await wait();
    expect((await p)?.id).toBe(7_001);
    expect(offsets(calls).filter((o) => o === 7_000).length).toBe(1);
  });

  it("retries a failed request after the retry delay", async () => {
    const h = make(1000);
    h.failNext(true);
    h.s.setQuery("a");
    await wait(5);
    expect(h.s.total()).toBe(-1);
    h.failNext(false);
    await wait(60);
    expect(h.s.total()).toBe(1000);
  });

  it("an empty result is loaded and requests nothing more", async () => {
    const { s, calls } = make(0);
    s.setQuery("a");
    await wait();
    expect(s.total()).toBe(0);
    expect(s.rangeLoaded(0, 20)).toBe(true);
    const n = calls.length;
    s.setVisible(0, 20);
    expect(calls.length).toBe(n);
  });
});
