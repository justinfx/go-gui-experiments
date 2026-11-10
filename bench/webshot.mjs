// Screenshot only the web app's page (never the desktop): node bench/webshot.mjs <url> <out.png> <dpr>
import { chromium } from 'playwright';
const [url, out, dpr] = process.argv.slice(2);
const b = await chromium.launch({ channel: 'chrome', headless: false });
const p = await (await b.newContext({ viewport: { width: 1280, height: 800 }, deviceScaleFactor: Number(dpr || 2) })).newPage();
await p.goto(url); await p.waitForTimeout(8000);
await p.screenshot({ path: out }); await b.close();
