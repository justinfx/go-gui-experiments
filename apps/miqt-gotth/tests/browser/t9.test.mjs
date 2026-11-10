// T9 (dirty guard) driven through a real Chrome with the DevTools protocol.
// Usage: node t9.test.mjs <base-url>
// Uses only Node built-ins (global fetch and WebSocket); no npm packages.
import { spawn } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const base = process.argv[2];
const chromePath = process.env.CHROME || "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const port = 9400 + Math.floor(Math.random() * 400);
const profile = mkdtempSync(join(tmpdir(), "gotth-chrome-"));
const chrome = spawn(chromePath, [
  "--headless=new", `--remote-debugging-port=${port}`, `--user-data-dir=${profile}`,
  "--no-first-run", "--no-default-browser-check", "--window-size=1280,800", "about:blank",
], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let failed = false;
function check(ok, msg) { if (!ok) { failed = true; console.error("FAIL: " + msg); } else console.log("ok: " + msg); }

async function main() {
  let targets;
  for (let i = 0; i < 50; i++) {
    try { targets = await (await fetch(`http://127.0.0.1:${port}/json`)).json(); if (targets.length) break; } catch { /* not up yet */ }
    await sleep(200);
  }
  const page = targets.find((t) => t.type === "page");
  const ws = new WebSocket(page.webSocketDebuggerUrl);
  await new Promise((r) => ws.addEventListener("open", r));
  let id = 0; const pending = new Map();
  ws.addEventListener("message", (m) => { const d = JSON.parse(m.data); if (d.id && pending.has(d.id)) { pending.get(d.id)(d); pending.delete(d.id); } });
  const send = (method, params = {}) => new Promise((res) => { const i = ++id; pending.set(i, res); ws.send(JSON.stringify({ id: i, method, params })); });
  const ev = async (expr) => { const r = await send("Runtime.evaluate", { expression: expr, awaitPromise: true, returnByValue: true }); if (r.result.exceptionDetails) throw new Error(JSON.stringify(r.result.exceptionDetails)); return r.result.result.value; };

  await send("Page.enable");
  await send("Page.navigate", { url: base + "/" });
  for (let i = 0; i < 50 && !(await ev("!!document.querySelector('#row-2') && !!window.htmx && !!window.templ")); i++) await sleep(200);

  const sel = (id) => ev(`document.querySelector('#detail').dataset.itemId`);
  await ev("document.querySelector('#row-2').click()"); await sleep(600);
  check((await sel()) === "2", "row 2 selected");
  await ev("(()=>{const n=document.querySelector('#f-name'); n.value='Changed by test'; n.dispatchEvent(new Event('input',{bubbles:true}));})()"); await sleep(300);
  check(await ev("!document.querySelector('#btn-save').disabled"), "Save enabled while dirty");
  check(await ev("!document.querySelector('#dirty-msg').classList.contains('hidden')"), "Unsaved changes text shown");
  await ev("document.querySelector('#row-3').click()"); await sleep(900);
  check(await ev("(document.querySelector('#confirm-dialog')?.innerText||'').includes('Discard unsaved changes?') && getComputedStyle(document.querySelector('#confirm-dialog')).display !== 'none' && !document.querySelector('#confirm-dialog').hidden"), "confirm dialog shown");
  check((await sel()) === "2", "selection unchanged while dialog is open");
  await ev("document.querySelector('#confirm-cancel').click()"); await sleep(600);
  check((await sel()) === "2" && (await ev("document.querySelector('#f-name').value")) === "Changed by test", "Cancel keeps selection and edits");
  await ev("document.querySelector('#row-3').click()"); await sleep(900);
  await ev("document.querySelector('#confirm-discard').click()"); await sleep(900);
  check((await sel()) === "3", "Discard moves the selection to row 3");
  ws.close();
}

try { await main(); } catch (e) { failed = true; console.error(e); }
chrome.kill();
process.exit(failed ? 1 : 0);
