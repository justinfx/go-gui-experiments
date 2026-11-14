import { describe, expect, it, vi } from "vitest";
import { Pager } from "./pager";

type Item = { id: number };
function make(total = 1000, delay = 0) {
  const calls: number[] = [];
  let gate: (() => void)[] = [];
  const fetcher = vi.fn(async (_q: string, offset: number, limit: number) => {
    calls.push(offset);
    if (delay < 0) await new Promise<void>((r) => gate.push(r));
    else await new Promise((r) => setTimeout(r, delay));
    return { items: Array.from({ length: Math.min(limit, total - offset) }, (_, i) => ({ id: offset + i + 1 })), total };
  });
  const changes = { n: 0 };
  const p = new Pager<Item, string>(fetcher, () => changes.n++, { retryMs: 20 });
  return { p, calls, fetcher, changes, release: () => gate.splice(0).forEach((g) => g()) };
}
const wait = (ms = 20) => new Promise((r) => setTimeout(r, ms));

describe("Pager", () => {
  it("requests only the first page while total is unknown", async () => {
    const { p, calls } = make();
    p.setQuery("a");
    p.setVisible(0, 30);
    expect(calls).toEqual([0]);
    expect(p.total()).toBe(-1);
    await wait();
    expect(p.total()).toBe(1000);
    expect(p.row(0)?.id).toBe(1);
    expect(p.row(199)?.id).toBe(200);
    expect(p.row(200)).toBeUndefined();
  });

  it("limits in-flight requests and orders by distance", async () => {
    const h = make(5000, -1);
    h.p.setQuery("a");
    h.release();
    await wait();
    h.release();
    await wait();
    h.p.setVisible(0, 10);
    h.release();
    await wait();
    h.calls.length = 0;
    h.p.setVisible(1000, 1600);
    expect(h.calls).toEqual([1000, 1200]); // anchor page 5 first, then nearest
    h.release();
    await wait();
    expect(h.p.stats().inFlight).toBeLessThanOrEqual(2);
  });

  it("discards responses from an older generation", async () => {
    const h = make(1000, -1);
    h.p.setQuery("a");
    h.p.setQuery("b");
    h.release();
    await wait();
    expect(h.calls.slice(0, 2)).toEqual([0, 0]);
    expect(h.p.total()).toBe(1000);
    expect(h.p.row(0)?.id).toBe(1);
  });

  it("evicts least recently touched non-visible pages over maxPages", async () => {
    const f = async (_q: string, offset: number, limit: number) => ({ items: Array.from({ length: limit }, (_, i) => ({ id: offset + i })), total: 100000 });
    const p = new Pager<Item, string>(f, () => {}, { maxPages: 3, prefetchMargin: 0 });
    p.setQuery("a");
    for (const start of [0, 200, 400, 600, 800]) {
      p.setVisible(start, start + 10);
      await wait(5);
    }
    expect(p.stats().cachedPages).toBeLessThanOrEqual(3);
    expect(p.row(800)).toBeDefined();
    expect(p.row(0)).toBeUndefined();
  });

  it("retries a failed request after the delay", async () => {
    let n = 0;
    const p = new Pager<Item, string>(
      async () => {
        if (n++ === 0) throw new Error("boom");
        return { items: [{ id: 1 }], total: 1 };
      },
      () => {},
      { retryMs: 30 },
    );
    p.setQuery("a");
    await wait(10);
    expect(p.total()).toBe(-1);
    await wait(80);
    expect(p.total()).toBe(1);
  });

  it("invalidate keeps visible data until the refetch arrives", async () => {
    let v = 1;
    const p = new Pager<{ v: number }, string>(async () => ({ items: [{ v }], total: 1 }), () => {});
    p.setQuery("a");
    await wait(5);
    expect(p.row(0)?.v).toBe(1);
    v = 2;
    p.invalidate();
    expect(p.row(0)?.v).toBe(1);
    await wait(5);
    expect(p.row(0)?.v).toBe(2);
  });

  it("reports rangeLoaded and an empty result as loaded", async () => {
    const p = new Pager<Item, string>(async () => ({ items: [], total: 0 }), () => {});
    p.setQuery("a");
    expect(p.rangeLoaded(0, 10)).toBe(false);
    await wait(5);
    expect(p.rangeLoaded(0, 10)).toBe(true);
  });
});
