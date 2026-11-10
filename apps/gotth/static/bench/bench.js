// GGUIX bench script (section 9). Loaded only in bench mode.
// Each step prints one `GGUIX_METRIC {...}` line to the console; in the
// desktop shell the line is also POSTed to /bench/metric so the Go server
// writes it to stdout.
(function () {
  "use strict";
  var G = window.GGUIX;
  var desktop = document.body.dataset.desktop === "true";
  var $ = function (s) { return document.querySelector(s); };
  var now = function () { return performance.now(); };
  var sleep = function (ms) { return new Promise(function (r) { setTimeout(r, ms); }); };

  function emit(event, fields) {
    var line = "GGUIX_METRIC " + JSON.stringify(Object.assign({ event: event }, fields || {}));
    console.log(line);
    if (desktop) return fetch("/bench/metric", { method: "POST", body: line, keepalive: true }).catch(function () {});
    return Promise.resolve();
  }

  window.addEventListener("error", function (e) { emit("jserror", { message: String(e.message), line: e.lineno, src: String(e.filename).slice(-30) }); });
  window.addEventListener("unhandledrejection", function (e) { emit("jserror", { message: "rejection " + String(e.reason && e.reason.message || e.reason) }); });

  function r1(x) { return Math.round(x * 10) / 10; }

  // Poll until every row in the visible range has data.
  function waitLoaded(timeoutMs) {
    var t0 = now();
    return new Promise(function (resolve, reject) {
      (function poll() {
        if (G.isLoaded()) return resolve();
        if (now() - t0 > (timeoutMs || 60000)) return reject(new Error("timeout waiting for rows"));
        setTimeout(poll, 2);
      })();
    });
  }

  // UI tick probe: a 16 ms timer; the delay is how late each tick ran.
  function probe() {
    var gaps = [], stopped = false, expected = now() + 16;
    (function tick() {
      if (stopped) return;
      var t = now();
      gaps.push(Math.max(0, t - expected));
      expected = t + 16;
      setTimeout(tick, 16);
    })();
    return {
      stop: function () {
        stopped = true;
        var s = gaps.slice().sort(function (a, b) { return a - b; });
        var max = s.length ? s[s.length - 1] : 0;
        var p95 = s.length ? s[Math.min(s.length - 1, Math.floor(s.length * 0.95))] : 0;
        return { max_gap_ms: r1(max), p95_gap_ms: r1(p95) };
      }
    };
  }

  // Steps every 16 ms; counts the steps at which a placeholder was visible
  // when the next step began.
  async function steps(n, fn) {
    var placeholders = 0, start = now();
    for (var k = 0; k < n; k++) {
      if (k > 0 && G.placeholderVisible()) placeholders++;
      fn(k);
      var due = start + (k + 1) * 16;
      var wait = due - now();
      if (wait > 0) await sleep(wait);
    }
    if (G.placeholderVisible()) placeholders++;
    return placeholders;
  }

  async function exportStep() {
    var rows = 0;
    var done = new Promise(function (resolve) {
      document.addEventListener("gguix:export", function h(e) {
        if (e.detail.state === "done") { rows = e.detail.rows; document.removeEventListener("gguix:export", h); resolve(); }
        else if (e.detail.state === "error" || e.detail.state === "cancelled") { document.removeEventListener("gguix:export", h); resolve(); }
      });
    });
    var p = probe();
    var t0 = now();
    await G.platform.startExport();
    await Promise.race([done, sleep(30000).then(function () { throw new Error("export did not finish"); })]);
    var ms = now() - t0;
    var g = p.stop();
    await emit("export", Object.assign({ rows: rows, ms: r1(ms) }, g));
  }

  // A step that fails (or times out) reports an error metric and the script
  // carries on with the next step, so the run always reaches "done".
  async function guarded(name, fn) {
    try {
      await fn();
    } catch (err) {
      await emit("error", { step: name, message: String(err && err.message || err) });
    }
  }

  async function run() {
    var t0, p, g, ph, total;
    await guarded("ready", async function () {
      await waitLoaded(20000);
      var readyMs = desktop && window.GGUIX_PROCESS_START ? Date.now() - window.GGUIX_PROCESS_START : now();
      await emit("ready", { ms: r1(readyMs) });
    });

    await guarded("jump", async function () {
      t0 = now();
      await G.gotoRow(500001);
      await waitLoaded(20000);
      await emit("jump", { ms: r1(now() - t0) });
    });

    await guarded("scroll", async function () {
      p = probe();
      t0 = now();
      ph = await steps(200, function (k) { G.scrollToRow(500000 + k * 25); });
      await waitLoaded(20000);
      total = now() - t0;
      g = p.stop();
      await emit("scroll", Object.assign({ total_ms: r1(total), placeholder_steps: ph }, g));
    });

    await guarded("cards_scroll", async function () {
      t0 = now();
      await G.setView("cards");
      await G.gotoRow(200001);
      await waitLoaded(20000);
      var sw = now() - t0;
      p = probe();
      t0 = now();
      ph = await steps(200, function (k) { G.scrollToRow(200000 + k * 10); });
      await waitLoaded(20000);
      total = now() - t0;
      g = p.stop();
      await emit("cards_scroll", Object.assign({ switch_ms: r1(sw), total_ms: r1(total), placeholder_steps: ph }, g));
    });
    await guarded("table", async function () {
      await G.setView("table");
      await waitLoaded(20000);
    });

    await guarded("sort", async function () {
      t0 = now();
      await G.sort("name", true);
      await waitLoaded(20000);
      await emit("sort", { ms: r1(now() - t0) });
    });

    await guarded("search", async function () {
      t0 = now();
      var before = $("#sb-count").textContent;
      G.setSearch("steel");
      await new Promise(function (resolve, reject) {
        (function poll() {
          var c = $("#sb-count").textContent;
          if (c !== before && G.isLoaded()) return resolve();
          if (now() - t0 > 20000) return reject(new Error("search did not finish"));
          setTimeout(poll, 2);
        })();
      });
      var cnt = parseInt($("#sb-count").textContent.replace(/[^0-9]/g, ""), 10);
      await emit("search", { ms: r1(now() - t0), total: cnt });
    });

    await guarded("export", exportStep);
    await emit("done", {});
  }

  if (document.readyState === "complete") setTimeout(run, 0);
  else window.addEventListener("load", function () { setTimeout(run, 0); });
})();
