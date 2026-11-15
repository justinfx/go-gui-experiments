// Capture the page (app only, no browser chrome) of a served web build.
// Usage: node bench/shot-web.mjs <url> <out.png> [timeoutMs]
// Waits until the status bar reports loaded rows ("rows N-M"), then 2 s more.
// Canvas-drawn apps have no DOM text, so after the timeout it captures anyway.
import { chromium } from 'playwright';

const [url, out, t = '30000'] = process.argv.slice(2);
const browser = await chromium.launch({ channel: 'chrome', headless: true });
const page = await browser.newPage({ viewport: { width: 1280, height: 800 }, deviceScaleFactor: 2 });
await page.goto(url);
try {
  await page.waitForFunction(() => /rows \d+-\d+/.test(document.body.innerText), null, { timeout: +t });
} catch {
  await page.waitForTimeout(8000);
}
await page.waitForTimeout(2000);
await page.screenshot({ path: out });
await browser.close();
