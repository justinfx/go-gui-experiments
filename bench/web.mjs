// Web bench driver: opens the app with ?bench=1 in Google Chrome (headed),
// collects GGUIX_METRIC console lines until "done", samples JS heap and DOM
// node counts through CDP, samples the RSS of Chrome's renderer and GPU
// processes (CDP SystemInfo.getProcessInfo for the pids, ps for RSS; this
// covers wasm linear memory, which the JS heap figure does not), and records
// the initial transfer size, DPR, viewport and Chrome version.
// Usage: node bench/web.mjs [url] [timeoutMs]   (prints JSON to stdout)
// GGUIX_ENGINE=webkit runs Playwright's WebKit instead (no CDP memory sampling;
// DOM nodes counted as elements). Used for the Chrome vs WebKit check only.
import { chromium, webkit } from 'playwright';
import { execFileSync } from 'node:child_process';

const url = process.argv[2] || 'http://localhost:8090/?bench=1';
const timeoutMs = Number(process.argv[3] || 300000);

const engine = process.env.GGUIX_ENGINE || 'chrome';
const browser = engine === 'webkit' ? await webkit.launch({ headless: false }) : await chromium.launch({ channel: 'chrome', headless: false });
const viewport = { width: 1280, height: 800 };
const dpr = Number(process.env.GGUIX_DPR || 2);
const context = await browser.newContext({ viewport, deviceScaleFactor: dpr });
const browserCdp = engine === 'webkit' ? null : await browser.newBrowserCDPSession();
const page = await context.newPage();
const cdp = engine === 'webkit' ? null : await context.newCDPSession(page);
if (cdp) await cdp.send('Performance.enable');

const metrics = [];
const consoleErrors = [];
let done;
const finished = new Promise((resolve) => (done = resolve));
page.on('console', (msg) => {
  const all = msg.text();
  for (const text of all.split('\n')) {
  if (text.startsWith('GGUIX_METRIC ')) {
    try {
      const m = JSON.parse(text.slice('GGUIX_METRIC '.length));
      metrics.push(m);
      if (m.event === 'done') done();
    } catch (e) {
      consoleErrors.push('bad metric: ' + text);
    }
  } else if (msg.type() === 'error' && text) {
    consoleErrors.push(text.slice(0, 300));
  }
  }
});
page.on('pageerror', (e) => consoleErrors.push('pageerror: ' + String(e).slice(0, 300)));

// RSS in KB of the given pids, from ps.
function rssKB(pids) {
  if (!pids.length) return 0;
  try {
    const out = execFileSync('ps', ['-o', 'rss=', '-p', pids.join(',')], { encoding: 'utf8' });
    return out.split(/\s+/).filter(Boolean).reduce((a, x) => a + Number(x), 0);
  } catch {
    return 0;
  }
}

let peakHeap = 0;
let peakNodes = 0;
let peakRenderer = 0;
let peakGpu = 0;
let rendererPids = [];
let memError = null;
const sampler = setInterval(async () => {
  if (!cdp) { try { peakNodes = Math.max(peakNodes, await page.evaluate(() => document.getElementsByTagName('*').length)); } catch {} return; }
  try {
    const { processInfo } = await browserCdp.send('SystemInfo.getProcessInfo');
    rendererPids = processInfo.filter((p) => p.type === 'renderer').map((p) => p.id);
    const gpu = processInfo.filter((p) => p.type.toLowerCase() === 'gpu').map((p) => p.id);
    peakRenderer = Math.max(peakRenderer, rssKB(rendererPids));
    peakGpu = Math.max(peakGpu, rssKB(gpu));
  } catch (e) {
    memError = String(e).slice(0, 200);
  }
  try {
    const { metrics: m } = await cdp.send('Performance.getMetrics');
    const get = (n) => (m.find((x) => x.name === n) || {}).value || 0;
    peakHeap = Math.max(peakHeap, get('JSHeapUsedSize'));
    peakNodes = Math.max(peakNodes, get('Nodes'));
  } catch {}
}, 250);

const t0 = Date.now();
let transferKB = null;
let timedOut = false;
try {
  await page.goto(url, { waitUntil: 'load', timeout: 120000 });
  const readyOrTimeout = new Promise((r) => setTimeout(r, timeoutMs, 'timeout'));
  const firstReady = new Promise((resolve) => {
    const iv = setInterval(() => {
      if (metrics.some((m) => m.event === 'ready' || m.event === 'error')) { clearInterval(iv); resolve(); }
    }, 50);
  });
  await Promise.race([firstReady, readyOrTimeout]);
  transferKB = await page.evaluate(() => {
    const nav = performance.getEntriesByType('navigation')[0];
    let total = nav ? nav.transferSize : 0;
    for (const r of performance.getEntriesByType('resource')) total += r.transferSize || 0;
    return Math.round(total / 1024);
  });
  const res = await Promise.race([finished.then(() => 'done'), readyOrTimeout]);
  timedOut = res === 'timeout';
} catch (e) {
  consoleErrors.push('driver: ' + String(e).slice(0, 300));
}
clearInterval(sampler);
const out = {
  url,
  wall_ms: Date.now() - t0,
  timed_out: timedOut,
  initial_transfer_kb: transferKB,
  engine,
  chrome_version: browser.version(),
  dpr,
  viewport,
  // JS heap only (CDP JSHeapUsedSize); excludes wasm linear memory.
  peak_js_heap_mb: Math.round((peakHeap / 1048576) * 10) / 10,
  // ps RSS summed over every renderer process (wasm memory lives here).
  peak_renderer_rss_mb: Math.round((peakRenderer / 1024) * 10) / 10,
  renderer_pids: rendererPids,
  // ps RSS of the GPU process (canvas and WebGL apps draw through it).
  peak_gpu_rss_mb: Math.round((peakGpu / 1024) * 10) / 10,
  memory_sampling_error: memError,
  peak_dom_nodes: peakNodes,
  metrics,
  console_errors: consoleErrors.slice(0, 20),
};
await browser.close();
// Exit once stdout is flushed; the readyOrTimeout timer would otherwise keep Node alive for timeoutMs.
process.stdout.write(JSON.stringify(out, null, 2) + '\n', () => process.exit(0));
