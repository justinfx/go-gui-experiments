// Runs the web bench in headless Chrome and prints the GGUIX_METRIC console lines.
// Usage: node bench-web.bench.mjs <base-url>   (the server must be running, e.g. make web)
import { spawn } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const base = process.argv[2] || "http://localhost:8090";
const chromePath = process.env.CHROME || "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const port = 9400 + Math.floor(Math.random() * 400);
const chrome = spawn(chromePath, ["--headless=new", `--remote-debugging-port=${port}`, `--user-data-dir=${mkdtempSync(join(tmpdir(), "gotth-chrome-"))}`, "--no-first-run", "--window-size=1280,800", "about:blank"], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let targets;
for (let i = 0; i < 50; i++) { try { targets = await (await fetch(`http://127.0.0.1:${port}/json`)).json(); if (targets.length) break; } catch { /* wait */ } await sleep(200); }
const ws = new WebSocket(targets.find((t) => t.type === "page").webSocketDebuggerUrl);
await new Promise((r) => ws.addEventListener("open", r));
let id = 0, done = false;
const send = (method, params = {}) => ws.send(JSON.stringify({ id: ++id, method, params }));
ws.addEventListener("message", (m) => {
  const d = JSON.parse(m.data);
  if (d.method === "Runtime.consoleAPICalled") {
    const text = d.params.args.map((a) => a.value ?? "").join(" ");
    if (text.startsWith("GGUIX_METRIC")) { console.log(text); if (text.includes('"event":"done"')) done = true; }
  }
});
send("Runtime.enable"); send("Page.enable");
send("Page.navigate", { url: base + "/?bench=1" });
for (let i = 0; i < 300 && !done; i++) await sleep(500);
chrome.kill();
process.exit(done ? 0 : 1);
