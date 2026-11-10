// GGUIX GoTTH client script (shared by the web and desktop targets).
// Server-rendered HTML does the work; this file adds the windowed scrolling
// helpers, the dirty guard, shortcuts and export status handling.
(function () {
  "use strict";
  var G = (window.GGUIX = window.GGUIX || {});
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };

  htmx.config.defaultSettleDelay = 0;
  htmx.config.defaultSwapDelay = 0;
  htmx.config.scrollIntoViewOnBoost = false;

  var isBench = !!window.GGUIX_BENCH;
  G.platform = G.platform || {};

  // ------------------------------------------------------------ small helpers
  function scroller() { return $("#scroller"); }
  function flash(msg, ms) {
    var el = $("#sb-msg");
    if (!el) return;
    el.textContent = msg;
    clearTimeout(flash.t);
    flash.t = setTimeout(function () { el.textContent = ""; }, ms || 2000);
  }
  G.flash = flash;
  function openDialog(triggerID) { var t = document.getElementById(triggerID); if (t) t.click(); }
  function showError(msg) {
    var d = $("#error-text");
    if (d) d.textContent = msg;
    openDialog("error-trigger");
  }
  G.showError = showError;

  // ------------------------------------------------------------ list window
  // The list is a virtual scroller over server-rendered page blocks.
  //
  //   #scroller  native scroll container
  //     #vs      one box as tall as the whole result (capped, see scale below)
  //       .pg    a page block of 200 rows/cards, absolutely positioned
  //       .pg.ph a placeholder block ("…" rows) that is also the htmx element
  //              that fetches the page: hx-get=/page?...&p=N
  //
  // sync() runs on every scroll event. It works out which rows are visible
  // from scrollTop alone (rows have a fixed pitch, nothing is measured),
  // positions the blocks, and asks for the pages it wants: visible ones first,
  // then two ahead in the scrolling direction and one behind, at most
  // FETCH_PARALLEL at a time. Placeholders that stop being wanted are aborted,
  // so only the latest viewport is ever being fetched.
  var PAGE = 200, MAX_LOADED = 6, FETCH_PARALLEL = 4, RETRY_MS = 1000;
  var V = null; // state of the current region, rebuilt by initScroller

  G.rowEl = function (i) { var s = scroller(); return s && s.querySelector('[data-row="' + i + '"]'); };

  function lastPageIdx() { return Math.max(0, Math.ceil(V.total / PAGE) - 1); }

  // Scroll position in "row space" (px from the first row). The browser caps
  // the height of an element, so a result taller than maxPx is scrolled
  // through a space scaled by V.scale; rows still have their real pitch.
  function topPx() {
    var st = V.s.scrollTop;
    if (V.snap && Math.abs(st - V.snap.st) < 1) return V.snap.px;
    return st / V.scale;
  }
  function visibleRange(tp) {
    var first = Math.max(0, Math.floor(tp / V.rowH));
    var last = Math.min(V.total - 1, Math.ceil((tp + V.vh) / V.rowH) - 1);
    return [first, Math.max(first, last)];
  }
  function setScale() {
    V.vh = V.s.clientHeight;
    var totalPx = V.total * V.rowH;
    V.scale = totalPx <= V.maxPx ? 1 : (V.maxPx - V.vh) / (totalPx - V.vh);
  }

  G.scrollToRow = function (i) {
    if (!V || V.s !== scroller() || !V.vs) return;
    i = Math.max(0, Math.min(V.total - 1, i));
    var px = i * V.rowH;
    var maxST = V.vs.offsetHeight - V.vh;
    var st = Math.min(px * V.scale, Math.max(0, maxST));
    V.s.scrollTop = st;
    V.snap = Math.abs(V.s.scrollTop - px * V.scale) < 1 ? { st: V.s.scrollTop, px: px } : null;
    sync();
  };

  // A placeholder block is as tall as its page but only carries "…" rows for
  // the part that is on screen (a prefetch placeholder carries none), so
  // scrolling quickly past many pages stays cheap.
  function fillPlaceholder(el, p, vr) {
    var start = p * PAGE, end = Math.min(V.total, start + PAGE) - 1;
    var a = Math.max(vr[0] - 1, start), b = Math.min(vr[1] + 1, end);
    var have = el.dataset.rows ? el.dataset.rows.split(",") : null;
    if (have && a >= +have[0] && b <= +have[1]) return;
    var host = V.view === "cards" ? el : el.tbody;
    host.textContent = "";
    el.dataset.rows = "";
    if (a > b) { el.dataset.off = "0"; return; }
    var row = $("#ph-tpl", V.vs).content.firstElementChild;
    var frag = document.createDocumentFragment();
    for (var i = a; i <= b; i++) frag.appendChild(row.cloneNode(true));
    host.appendChild(frag);
    el.dataset.rows = a + "," + b;
    el.dataset.off = String((a - start) * V.rowH);
  }

  function makePlaceholder(p) {
    var tpl = $("#ph-tpl", V.vs);
    if (!tpl || !tpl.content.firstElementChild) return null;
    var el = document.createElement("div");
    if (V.view === "cards") {
      el.className = "pg ph px-2";
      el.style.cssText = "position:absolute;left:0;right:0";
    } else {
      el.className = "pg ph";
      el.style.cssText = "position:absolute;left:0;width:" + V.vs.style.width;
      var tb = document.createElement("table");
      tb.className = "caption-bottom border-separate border-spacing-0 text-sm";
      tb.setAttribute("data-slot", "table");
      tb.style.cssText = "table-layout:fixed;width:" + V.vs.style.width;
      var cg = $("#thead-wrap colgroup");
      if (cg) tb.appendChild(cg.cloneNode(true));
      el.tbody = document.createElement("tbody");
      tb.appendChild(el.tbody);
      el.appendChild(tb);
    }
    el.dataset.page = String(p);
    el.setAttribute("hx-get", V.url + p);
    el.setAttribute("hx-trigger", "gguix-load");
    el.setAttribute("hx-swap", "outerHTML");
    el.setAttribute("hx-sync", "this:drop");
    fillPlaceholder(el, p, V.visible || visibleRange(topPx()));
    V.vs.appendChild(el);
    htmx.process(el);
    return el;
  }

  function reindex() {
    V.blocks = {};
    $$(".pg", V.vs).forEach(function (b) { V.blocks[b.dataset.page] = b; });
  }

  // Pages to have, most important first.
  function wantedPages(vr) {
    var last = lastPageIdx();
    var p0 = Math.floor(vr[0] / PAGE), p1 = Math.floor(vr[1] / PAGE);
    var out = [], seen = {};
    function add(p) { if (p >= 0 && p <= last && !seen[p]) { seen[p] = 1; out.push(p); } }
    for (var p = p0; p <= p1; p++) add(p);
    if (V.dir >= 0) { add(p1 + 1); if (V.dir > 0) add(p1 + 2); add(p0 - 1); }
    else { add(p0 - 1); add(p0 - 2); add(p1 + 1); }
    return out;
  }

  function sync() {
    if (!V || V.s !== scroller() || !V.vs) return;
    var tp = topPx();
    var d = tp - V.lastTp;
    V.lastTp = tp;
    if (Math.abs(d) > V.rowH * 3) {
      V.dir = Math.abs(d) > PAGE * V.rowH * 10 ? 0 : (d > 0 ? 1 : -1);
      clearTimeout(V.dirTimer);
      V.dirTimer = setTimeout(function () { if (V) V.dir = 0; }, 300);
    }
    var vr = V.visible = visibleRange(tp);
    var want = wantedPages(vr), wantSet = {};
    want.forEach(function (p) { wantSet[p] = 1; });

    // Drop placeholders nobody wants any more (aborting their request).
    $$(".ph", V.vs).forEach(function (el) {
      if (wantSet[el.dataset.page]) return;
      if (el.dataset.fly) htmx.trigger(el, "htmx:abort");
      el.remove();
      delete V.blocks[el.dataset.page];
    });
    // Loaded pages far from the viewport go (the server still has them), and
    // at most MAX_LOADED stay: drop the ones farthest away.
    var near0 = Math.floor(vr[0] / PAGE) - 2, near1 = Math.floor(vr[1] / PAGE) + 3;
    $$(".pg:not(.ph)", V.vs).forEach(function (b) {
      var p = +b.dataset.page;
      if (p < near0 || p > near1) { b.remove(); delete V.blocks[p]; V.dirtyRows = true; }
    });
    var loaded = $$(".pg:not(.ph)", V.vs);
    if (loaded.length > MAX_LOADED) {
      var mid = (vr[0] + vr[1]) / 2 / PAGE;
      loaded.filter(function (b) { return !wantSet[b.dataset.page]; })
        .sort(function (a, b) { return Math.abs(+b.dataset.page - mid) - Math.abs(+a.dataset.page - mid); })
        .slice(0, loaded.length - MAX_LOADED)
        .forEach(function (b) { b.remove(); delete V.blocks[b.dataset.page]; V.dirtyRows = true; });
    }
    // Position the blocks, then make sure every wanted page has one.
    var off = V.scale === 1 ? 0 : Math.round(V.s.scrollTop - tp);
    function place(p, el) {
      if (el.classList.contains("ph")) fillPlaceholder(el, p, vr);
      var t = Math.round(p * PAGE * V.rowH + off + (+el.dataset.off || 0)) + "px";
      if (el.style.top !== t) el.style.top = t;
    }
    var flying = 0;
    want.forEach(function (p) {
      var el = V.blocks[p];
      if (!el) { el = V.blocks[p] = makePlaceholder(p); if (!el) return; }
      if (el.classList.contains("ph") && el.dataset.fly) flying++;
    });
    $$(".pg", V.vs).forEach(function (el) { place(+el.dataset.page, el); });
    // Start fetches in priority order.
    var now = Date.now();
    for (var i = 0; i < want.length && flying < FETCH_PARALLEL; i++) {
      var ph = V.blocks[want[i]];
      if (!ph || !ph.classList.contains("ph") || ph.dataset.fly) continue;
      if (ph.dataset.fail && now - +ph.dataset.fail < RETRY_MS) continue;
      ph.dataset.fly = "1";
      flying++;
      htmx.trigger(ph, "gguix-load");
    }
    if (V.dirtyRows) { V.dirtyRows = false; updateRowsStatus(); }
  }
  G.loadVisible = sync;

  // A "…" row or "Loading…" card is inside the visible range.
  G.placeholderVisible = function () {
    if (!V || V.s !== scroller() || !V.vs) return false;
    var vr = visibleRange(topPx());
    for (var p = Math.floor(vr[0] / PAGE); p <= Math.floor(vr[1] / PAGE); p++) {
      var b = V.blocks[p];
      if (!b || b.classList.contains("ph")) return true;
    }
    return false;
  };

  // Every row in the visible range has data (the refresh call has been made:
  // the page block is in the DOM).
  G.isLoaded = function () {
    var s = scroller();
    if (!s || !V || V.s !== s) return false;
    if (!V.vs) return !!s.querySelector('[data-slot="empty"]');
    return !G.placeholderVisible();
  };

  function updateRowsStatus() {
    var el = $("#sb-rows");
    var s = scroller();
    if (!el || !s) return;
    var pages = $$(".pg:not(.ph)", s).map(function (p) { return +p.dataset.page; });
    var total = +s.dataset.total || 0;
    if (!pages.length || !total) { el.textContent = ""; return; }
    var a = Math.min.apply(null, pages) * PAGE + 1;
    var b = Math.min(total, (Math.max.apply(null, pages) + 1) * PAGE);
    el.textContent = " · rows " + a + "-" + b;
  }

  function initScroller() {
    var s = scroller();
    if (!s) return;
    document.body.dataset.view = s.dataset.view;
    if (V) clearTimeout(V.dirTimer);
    var vs = $("#vs", s);
    V = { s: s, vs: vs, view: s.dataset.view, total: +s.dataset.total || 0, rowH: +s.dataset.rowH, maxPx: +s.dataset.maxPx,
      url: s.dataset.pageUrl, blocks: {}, snap: null, dir: 0, lastTp: 0, vh: 0, scale: 1 };
    if (vs) {
      setScale();
      reindex();
      s.addEventListener("scroll", function () {
        var h = $("#thead-wrap");
        if (h) h.scrollLeft = s.scrollLeft;
        sync();
      }, { passive: true });
      var r = +s.dataset.scrollRow;
      if (r >= 0) G.scrollToRow(r); else { s.scrollTop = 0; sync(); }
      markSelected(selectedId());
    }
    updateRowsStatus();
    document.dispatchEvent(new CustomEvent("gguix:region"));
  }
  window.addEventListener("resize", function () { if (V && V.vs) { setScale(); sync(); } });

  // Resolves when the next list region has been swapped in. htmx.ajax's own
  // promise did not always settle in WKWebView, so callers wait for this.
  function whenRegion() {
    return new Promise(function (resolve) {
      var done = false;
      function fin() {
        if (done) return;
        done = true;
        document.removeEventListener("gguix:region", fin);
        resolve();
      }
      document.addEventListener("gguix:region", fin);
      setTimeout(fin, 15000);
    });
  }

  // Page fetches are plain htmx requests from the placeholder blocks.
  document.addEventListener("htmx:afterRequest", function (e) {
    var el = e.detail.elt;
    if (!el || !el.classList || !el.classList.contains("ph")) return;
    delete el.dataset.fly;
    if (!e.detail.successful && el.isConnected) {
      // Failed: ask again after a second (an abort removes the element).
      el.dataset.fail = String(Date.now());
      setTimeout(sync, RETRY_MS + 20);
    }
  });
  document.addEventListener("htmx:afterSwap", function (e) {
    var cfg = e.detail.requestConfig;
    if (!cfg || !/^\/page\?/.test(cfg.path || "") || !V || !V.vs) return;
    reindex();
    V.dirtyRows = false;
    markSelected(selectedId());
    updateRowsStatus();
    sync();
  });

  // Full region swaps and form swaps are noticed with observers, so the same
  // code runs for the first render and for every htmx response.
  function observe(sel, fn) {
    var el = $(sel);
    if (!el) return;
    new MutationObserver(fn).observe(el, { childList: true });
  }

  // ------------------------------------------------------------ selection
  function selectedId() {
    var f = $("#detail");
    return f ? parseInt(f.dataset.itemId || "0", 10) || 0 : 0;
  }
  // The row index of the selection: from the DOM when its row is loaded,
  // otherwise the index remembered when it was selected (go-to-row, arrows).
  var selRow = -1;
  function selectedRowIdx() {
    var s = scroller();
    var el = s && s.querySelector('[data-state="selected"]');
    if (el) return el.dataset.row;
    return selRow >= 0 && selectedId() ? String(selRow) : undefined;
  }
  G.selectedRowIdx = selectedRowIdx;

  function markSelected(id) {
    var s = scroller();
    if (!id) selRow = -1;
    if (!s) return;
    $$('[data-state="selected"]', s).forEach(function (el) {
      el.removeAttribute("data-state");
      el.removeAttribute("aria-selected");
    });
    if (!id) return;
    $$('[data-id="' + id + '"]', s).forEach(function (el) {
      el.setAttribute("data-state", "selected");
      el.setAttribute("aria-selected", "true");
      selRow = +el.dataset.row;
    });
  }

  function selectItem(id) {
    if (!id || id === selectedId()) return;
    guard(function () {
      htmx.ajax("POST", "/select", { values: { id: id }, target: "#detail", swap: "outerHTML" });
    });
  }
  G.selectItem = selectItem;

  document.addEventListener("click", function (e) {
    var row = e.target.closest && e.target.closest("#scroller [data-id]");
    if (row) {
      var sc = scroller();
      if (sc) sc.focus({ preventScroll: true });
      selectItem(parseInt(row.dataset.id, 10));
    }
  });

  // Up/Down move the selection by row index. A row that is not in the DOM
  // yet is selected through /goto, which asks the server for that one row.
  document.addEventListener("keydown", function (e) {
    var s = scroller();
    if (!s || document.activeElement !== s || !V || !V.vs) return;
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    e.preventDefault();
    var cur = s.querySelector('[data-state="selected"]');
    var vr = visibleRange(topPx());
    var n = cur ? +cur.dataset.row + (e.key === "ArrowDown" ? 1 : -1) : vr[0];
    n = Math.max(0, Math.min(V.total - 1, n));
    var tp = topPx();
    if (n * V.rowH < tp) G.scrollToRow(n);
    else if ((n + 1) * V.rowH > tp + V.vh) G.scrollToRow(Math.ceil(((n + 1) * V.rowH - V.vh) / V.rowH));
    var el = G.rowEl(n);
    if (el) selectItem(parseInt(el.dataset.id, 10));
    else guard(function () { selRow = n; postGoto(n); });
  });

  // ------------------------------------------------------------ dirty guard
  var initialSnap = null;
  var pending = null;

  function snapshot(f) {
    var out = [];
    new FormData(f).forEach(function (v, k) { if (k !== "id" && k !== "rowidx") out.push(k + "=" + v); });
    return out.join("\u0001");
  }
  function isDirty() {
    var f = $("#detail");
    if (!f || !selectedId()) return false;
    if (f.dataset.forceDirty === "true") return true;
    return initialSnap !== null && snapshot(f) !== initialSnap;
  }
  G.isDirty = isDirty;

  function setDisabled(el, on) {
    if (!el) return;
    el.disabled = on;
    if (on) el.setAttribute("disabled", ""); else el.removeAttribute("disabled");
  }

  function refreshDirty() {
    var f = $("#detail");
    if (!f) return;
    var d = isDirty();
    setDisabled($("#btn-save", f), !d);
    setDisabled($("#btn-revert", f), !d);
    var m = $("#dirty-msg", f);
    if (m) m.classList.toggle("hidden", !d);
  }

  function initForm() {
    var f = $("#detail");
    if (!f) return;
    initialSnap = null;
    var id = selectedId();
    markSelected(id);
    refreshDirty();
    setTimeout(function () {
      initialSnap = snapshot(f);
      refreshDirty();
    }, 30);
  }

  function guard(fn) {
    if (isDirty()) {
      pending = fn;
      openDialog("confirm-trigger");
    } else {
      fn();
    }
  }
  G.guard = guard;

  document.addEventListener("click", function (e) {
    if (e.target.closest("#confirm-discard")) {
      var fn = pending; pending = null;
      if (fn) fn();
    } else if (e.target.closest("#confirm-cancel")) {
      pending = null;
    }
  });

  ["input", "change"].forEach(function (t) {
    document.addEventListener(t, function (e) {
      if (e.target.closest && e.target.closest("#detail")) setTimeout(refreshDirty, 0);
    });
  });
  document.addEventListener("slider-change", function (e) {
    var f = e.target.closest && e.target.closest("#detail");
    if (!f) return;
    var v = e.detail && e.detail.values && e.detail.values[0];
    var lbl = $('[data-slot="rating-label"]', f);
    if (lbl && v !== undefined) lbl.textContent = (Math.round(v * 10) / 10).toFixed(1);
    setTimeout(refreshDirty, 0);
  });

  function fmtDate(d) {
    if (!d) return "";
    if (typeof d === "string") return d.slice(0, 10);
    var m = String(d.getMonth() + 1).padStart(2, "0"), day = String(d.getDate()).padStart(2, "0");
    return d.getFullYear() + "-" + m + "-" + day;
  }
  function setRestock(v) {
    var inp = $("#f-restock");
    if (!inp) return;
    inp.value = v;
    var lbl = $('[data-slot="restock-label"]');
    if (lbl) lbl.textContent = v || "Pick a date";
    setTimeout(refreshDirty, 0);
  }
  document.addEventListener("calendar-select", function (e) {
    if (!e.target.closest || !e.target.closest("[data-restock-calendar]")) return;
    setRestock(fmtDate(e.detail && e.detail.selected));
  });

  document.addEventListener("htmx:configRequest", function (e) {
    if (e.detail.path === "/save") {
      var idx = selectedRowIdx();
      if (idx !== undefined) e.detail.parameters["rowidx"] = idx;
    }
  });

  // ------------------------------------------------------------ toolbar
  function filtersChanged() { htmx.trigger("#filters", "filters-changed"); }
  G.filtersChanged = filtersChanged;
  var searchTimer;
  var suppress = false;

  document.addEventListener("input", function (e) {
    if (e.target.id === "search") {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(filtersChanged, isBench ? 0 : 250);
    }
  });
  document.addEventListener("change", function (e) {
    var t = e.target;
    if (t.id === "status" || (t.name === "cat" && t.form && t.form.id === "filters")) {
      if (!suppress) filtersChanged();
    } else if (t.id === "theme") {
      G.setTheme(t.value);
    }
  });

  G.setSearch = function (text) {
    var inp = $("#search");
    clearTimeout(searchTimer);
    inp.value = text;
    filtersChanged();
  };

  G.setTheme = function (t) {
    document.documentElement.dataset.theme = t;
    if (window.__applyTheme) window.__applyTheme();
    var sel = $("#theme");
    if (sel && sel.value !== t) sel.value = t;
    $$('[data-menu-group="theme"] [data-slot="menubar-radio-item"]').forEach(function (it) {
      var on = it.dataset.templValue === t;
      it.toggleAttribute("data-checked", on);
      it.toggleAttribute("data-unchecked", !on);
      it.setAttribute("aria-checked", String(on));
    });
    fetch("/theme", { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: "theme=" + encodeURIComponent(t) });
  };

  G.setView = function (v) {
    if (document.body.dataset.view === v && scroller() && scroller().dataset.view === v) return Promise.resolve();
    var vals = { view: v };
    var idx = selectedRowIdx();
    if (idx !== undefined) vals.row = idx;
    $$('[data-menu-group="view"] [data-slot="menubar-radio-item"]').forEach(function (it) {
      var on = it.dataset.templValue === v;
      it.toggleAttribute("data-checked", on);
      it.toggleAttribute("data-unchecked", !on);
      it.setAttribute("aria-checked", String(on));
    });
    var done = whenRegion();
    htmx.ajax("POST", "/view", { values: vals, target: "#list-region", swap: "innerHTML" });
    return done;
  };

  // Go to row: the list scrolls locally (the total is known, rows have a fixed
  // pitch) while /goto asks the server for the form of that one row; the
  // server answers from the page the list is fetching at the same moment.
  function postGoto(idx) {
    return new Promise(function (resolve) {
      var t = setTimeout(fin, 15000);
      function fin() { document.removeEventListener("htmx:afterRequest", h); clearTimeout(t); resolve(); }
      function h(e) { var c = e.detail.requestConfig; if (c && c.path === "/goto") fin(); }
      document.addEventListener("htmx:afterRequest", h);
      htmx.ajax("POST", "/goto", { values: { row: idx }, target: "#detail", swap: "outerHTML" });
    });
  }
  G.gotoRow = function (n) {
    if (!V || !V.vs || !V.total) return Promise.resolve();
    var idx = Math.max(0, Math.min(V.total - 1, (parseInt(n, 10) || 1) - 1));
    G.scrollToRow(idx);
    selRow = idx;
    return postGoto(idx);
  };

  document.addEventListener("submit", function (e) {
    if (e.target.id !== "goto") return;
    e.preventDefault();
    var v = $("#goto-row", e.target).value;
    guard(function () { G.gotoRow(v); });
  });

  G.sort = function (field, desc) {
    var vals = { sort: field };
    var cur = $('th[data-sort="' + field + '"]');
    var active = cur && /[▲▼]/.test(cur.textContent);
    var form = $("#filters");
    var data = {};
    new FormData(form).forEach(function (v, k) { data[k] = v; });
    var done = whenRegion();
    htmx.ajax("POST", "/query", { values: Object.assign(data, vals, { dir: desc ? "desc" : "asc", filters: "1" }), target: "#list-region", swap: "innerHTML" });
    return done;
  };

  G.reload = function () { filtersChanged(); };

  document.addEventListener("click", function (e) {
    var b = e.target.closest && e.target.closest("[data-action]");
    if (!b || b.disabled) return;
    var a = b.dataset.action;
    if (a === "view") G.setView(b.dataset.view);
    else if (a === "clear-categories") {
      suppress = true;
      $$('input[name="cat"]:checked').forEach(function (i) { i.click(); });
      suppress = false;
      filtersChanged();
    } else if (a === "restock-clear") setRestock("");
    else if (a === "save") htmx.trigger("#detail", "submit");
    else if (a === "revert") {
      var id = selectedId();
      pending = function () { htmx.ajax("GET", "/form?id=" + id, { target: "#detail", swap: "outerHTML" }); };
      openDialog("confirm-trigger");
    } else if (a === "copy-row") G.actions["copy-row"]();
    else if (a === "import-notes") G.actions["import-notes"]();
    else if (a === "export") G.actions.export();
    else if (a === "export-cancel") fetch("/export/cancel", { method: "POST" });
    // gguix:target desktop
    else if (a === "reveal-config") fetch("/host/reveal-config", { method: "POST" });
    else if (a === "reveal-export") fetch("/host/reveal-export", { method: "POST" });
    // gguix:end
  });

  // ------------------------------------------------------------ actions
  G.actions = {
    "export": function () {
      var btn = $("#export-btn");
      if (btn && btn.disabled) return;
      G.platform.startExport && G.platform.startExport();
    },
    "import-notes": function () {
      if (!selectedId()) return;
      Promise.resolve(G.platform.importNotes && G.platform.importNotes()).then(function (r) {
        if (!r) return;
        if (r.error) { showError(r.error); return; }
        var n = $("#f-notes");
        if (n) {
          n.value = Array.from(r.text || "").slice(0, 500).join("");
          n.dispatchEvent(new Event("input", { bubbles: true }));
        }
      });
    },
    "copy-row": function () {
      var f = $("#detail");
      if (!f || !selectedId()) return;
      var csv = f.dataset.csv || "";
      Promise.resolve(G.platform.copyText && G.platform.copyText(csv)).then(function () { flash("Row copied", 2000); });
    },
    "view-table": function () { G.setView("table"); },
    "view-cards": function () { G.setView("cards"); },
    "theme-system": function () { G.setTheme("system"); },
    "theme-light": function () { G.setTheme("light"); },
    "theme-dark": function () { G.setTheme("dark"); },
    "reload": function () { G.reload(); },
    "focus-search": function () { var s = $("#search"); if (s) { s.focus(); s.select(); } },
    "host": function () { var b = $("#host-btn"); if (b) b.click(); },
    "about": function () { openDialog("about-trigger"); },
    "reveal-export": function () { fetch("/host/reveal-export", { method: "POST" }); }
    // gguix:target desktop
    , "quit": function () { fetch("/native/quit", { method: "POST" }); }
    // gguix:end
  };

  document.addEventListener("click", function (e) {
    var it = e.target.closest && e.target.closest("[data-menu-action]");
    if (!it || it.hasAttribute("data-disabled")) return;
    var fn = G.actions[it.dataset.menuAction];
    if (fn) fn();
  });

  document.addEventListener("keydown", function (e) {
    if (!(e.metaKey || e.ctrlKey) || e.altKey) return;
    var k = e.key.toLowerCase();
    var map = { e: "export", o: "import-notes", "1": "view-table", "2": "view-cards", r: "reload", f: "focus-search", i: "host" };
    var a = null;
    if (e.shiftKey && k === "c") a = "copy-row";
    else if (!e.shiftKey && map[k]) a = map[k];
    // gguix:target desktop
    else if (!e.shiftKey && k === "q" && document.body.dataset.desktop === "true") a = "quit";
    // gguix:end
    if (!a) return;
    e.preventDefault();
    G.actions[a]();
  });

  // ------------------------------------------------------------ export status
  var baseline, handled = null;
  function onExportMarker() {
    var m = $("#export-marker");
    if (!m) return;
    var st = m.dataset.state, job = parseInt(m.dataset.job, 10) || 0;
    var running = st === "running";
    var btn = $("#export-btn");
    if (btn) setDisabled(btn, running);
    $$('[data-menu-action="export"]').forEach(function (it) {
      it.toggleAttribute("data-disabled", running);
      it.setAttribute("aria-disabled", String(running));
    });
    var terminal = st === "done" || st === "cancelled" || st === "error";
    if (baseline === undefined) {
      baseline = job;
      if (terminal) handled = job;
    }
    if (st === "done") {
      $$('[data-menu-action="reveal-export"]').forEach(function (it) { it.removeAttribute("data-disabled"); it.removeAttribute("aria-disabled"); });
    }
    if (terminal && job !== handled) {
      handled = job;
      var rows = parseInt(m.dataset.rows, 10) || 0;
      if (st === "done") {
        G.platform.notify && G.platform.notify("Export complete", rows.toLocaleString("en-US") + " rows written");
      } else if (st === "cancelled") {
        flash("Export cancelled", 2000);
      } else {
        showError(m.dataset.err || "Export failed");
      }
    }
    document.dispatchEvent(new CustomEvent("gguix:export", { detail: { state: st, job: job, rows: parseInt(m.dataset.rows, 10) || 0 } }));
  }
  G.exportState = function () { var m = $("#export-marker"); return m ? m.dataset.state : "idle"; };

  // ------------------------------------------------------------ boot
  function boot() {
    var es = $("#export-status");
    if (es) new MutationObserver(onExportMarker).observe(es, { childList: true, subtree: true });
    observe("#list-region", initScroller);
    observe("#detail-pane", initForm);
    initScroller();
    initForm();
    var sync = $("#theme");
    if (sync) sync.value = document.documentElement.dataset.theme;
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", boot); else boot();
})();
