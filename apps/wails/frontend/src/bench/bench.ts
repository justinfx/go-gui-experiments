// Bench script (BRIEF section 9). Runs in the page: the desktop webview
// (GGUIX_BENCH=1) or a browser opened with ?bench=1.
import { tick } from "svelte";
import { app } from "$lib/state.svelte";
import { api } from "$lib/api";

const STEP_TIMEOUT = 60_000;
const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));
const now = () => performance.now();

function emit(event: string, fields: Record<string, unknown>): void {
  console.log("GGUIX_METRIC " + JSON.stringify({ event, ...fields }));
  if (app.isDesktop) void api.EmitMetric(event, JSON.stringify(fields));
}

/** Every row or card in the visible range has data and the DOM shows it. */
function loaded(): boolean {
  const h = app.active;
  const t = app.pager.total();
  if (!h || t < 0) return false;
  const [f, l] = h.visibleRange();
  if (!app.pager.rangeLoaded(f, l)) return false;
  const el = h.element();
  if (!el) return false;
  if (el.querySelector("[data-placeholder]")) return false;
  return t === 0 || el.querySelector("[data-index],[data-id]") !== null;
}

async function waitLoaded(cond: () => boolean = () => true): Promise<void> {
  const t0 = now();
  for (;;) {
    await tick();
    if (loaded() && cond()) return;
    if (now() - t0 > STEP_TIMEOUT) throw new Error("timeout");
    await sleep(4);
  }
}

class Probe {
  gaps: number[] = [];
  private on = false;
  start(): void {
    this.gaps = [];
    this.on = true;
    const loop = () => {
      if (!this.on) return;
      const due = now() + 16;
      setTimeout(() => {
        this.gaps.push(Math.max(0, now() - due));
        loop();
      }, 16);
    };
    loop();
  }
  stop(): { max_gap_ms: number; p95_gap_ms: number } {
    this.on = false;
    const g = [...this.gaps].sort((a, b) => a - b);
    const p95 = g.length ? g[Math.min(g.length - 1, Math.floor(g.length * 0.95))] : 0;
    return { max_gap_ms: round(g.length ? g[g.length - 1] : 0), p95_gap_ms: round(p95) };
  }
}
const round = (x: number) => Math.round(x * 100) / 100;

async function scrollSteps(start: number, stride: number): Promise<number> {
  let placeholder = 0;
  const t0 = now();
  for (let k = 0; k < 200; k++) {
    app.active?.scrollToRow(start + k * stride);
    // Absolute deadline, as in core/bench and gotth bench.js, so slow steps do not stretch the schedule.
    const due = t0 + (k + 1) * 16;
    const wait = due - now();
    if (wait > 0) await sleep(wait);
    else await tick();
    const h = app.active;
    if (h) {
      const [f, l] = h.visibleRange();
      if (!app.pager.rangeLoaded(f, l) || h.element()?.querySelector("[data-placeholder]")) placeholder++;
    }
  }
  return placeholder;
}

export async function runBench(): Promise<void> {
  try {
    await waitLoaded();
    const info = app.info!;
    const ready = info.target === "desktop" ? Date.now() - info.processStartMs : now();
    emit("ready", { ms: round(ready) });

    let t0 = now();
    app.goToRow(500_001);
    await waitLoaded();
    emit("jump", { ms: round(now() - t0) });

    const probe = new Probe();
    probe.start();
    t0 = now();
    let ph = await scrollSteps(500_000, 25);
    await waitLoaded();
    emit("scroll", { total_ms: round(now() - t0), ...probe.stop(), placeholder_steps: ph });

    t0 = now();
    app.setView("cards");
    await waitLoaded(() => app.active?.element()?.dataset.testid === "cards");
    app.goToRow(200_001);
    await waitLoaded();
    const switchMs = now() - t0;
    probe.start();
    t0 = now();
    ph = await scrollSteps(200_000, 10);
    await waitLoaded();
    emit("cards_scroll", { switch_ms: round(switchMs), total_ms: round(now() - t0), ...probe.stop(), placeholder_steps: ph });
    app.setView("table");
    await waitLoaded(() => app.active?.element()?.dataset.testid === "table");

    t0 = now();
    app.setSort("name", true);
    await waitLoaded();
    emit("sort", { ms: round(now() - t0) });

    t0 = now();
    app.searchInput = "steel";
    app.commitSearch();
    await waitLoaded();
    emit("search", { ms: round(now() - t0), total: app.pager.total() });

    probe.start();
    t0 = now();
    await app.startExport();
    while (app.exportRunning) await sleep(10);
    emit("export", { rows: app.lastExportRows, ms: round(now() - t0), ...probe.stop() });
  } catch (e) {
    emit("error", { message: String(e) });
  }
  emit("done", {});
  if (app.isDesktop) await api.Quit();
}
