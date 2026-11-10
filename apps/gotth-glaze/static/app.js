// GGUIX GoTTH client script (shared by the web and desktop targets).
// Server-rendered HTML does the work; this file adds the windowed scrolling
// helpers, the dirty guard, shortcuts and export status handling.
(function () {
  "use strict";
  var G = (window.GGUIX = window.GGUIX || {});
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };
  var MAX_LOADED = 5;

  htmx.config.defaultSettleDelay = 0;
  htmx.config.defaultSwapDelay = 0;
  htmx.config.scrollIntoViewOnBoost = false;

  var isBench = !!window.GGUIX_BENCH;
  var inflight = 0;
  G.platform = G.platform || {};

  // ------------------------------------------------------------ small helpers
  function scroller() { return $("#scroller"); }
  function headerHeight(s) {
    var th = s && s.querySelector("thead");
    return th ? th.getBoundingClientRect().height : 0;
  }
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
  G.rowEl = function (i) { var s = scroller(); return s && s.querySelector('[data-row="' + i + '"]'); };

  G.scrollToRow = function (i) {
    var s = scroller();
    if (!s) return;
    var el = G.rowEl(i);
    var hdr = headerHeight(s);
    if (!el) {
      // Not in the DOM window: go to the nearest end of the loaded window.
      var rows = $$("[data-row]", s);
      if (!rows.length) return;
      var first = +rows[0].dataset.row, last = +rows[rows.length - 1].dataset.row;
      if (i < first) { s.scrollTop = 0; } else { s.scrollTop = s.scrollHeight; }
      G.loadVisible();
      return;
    }
    var top = el.getBoundingClientRect().top - s.getBoundingClientRect().top + s.scrollTop - hdr;
    s.scrollTop = top;
    G.loadVisible();
  };

  // A placeholder (sentinel or spacer) is inside the visible range.
  G.placeholderVisible = function () {
    var s = scroller();
    if (!s) return false;
    var r = s.getBoundingClientRect(), hdr = headerHeight(s);
    return $$(".sentinel, .sp", s).some(function (el) {
      var b = el.getBoundingClientRect();
      return b.bottom > r.top + hdr && b.top < r.bottom;
    });
  };

  G.loadState = function () {
    var s = scroller();
    return { inflight: inflight, ph: G.placeholderVisible(), hasId: !!(s && s.querySelector("[data-id]")), view: s && s.dataset.view, items: s ? $$(".pg,.sp,.sentinel", s).map(function (e) { return e.className.split(" ")[0] + (e.dataset.page || e.dataset.dir); }).join(",") : "" };
  };

  G.isLoaded = function () {
    var s = scroller();
    if (!s || inflight > 0) return false;
    if (!s.querySelector("[data-id]") && !s.querySelector('[data-slot="empty"]')) return false;
    return !G.placeholderVisible();
  };

  function updateRowsStatus() {
    var el = $("#sb-rows");
    var s = scroller();
    if (!el || !s) return;
    var pages = $$(".pg", s).map(function (p) { return +p.dataset.page; });
    var total = +s.dataset.total || 0;
    if (!pages.length || !total) { el.textContent = ""; return; }
    var a = Math.min.apply(null, pages) * 200 + 1;
    var b = Math.min(total, (Math.max.apply(null, pages) + 1) * 200);
    el.textContent = " · rows " + a + "-" + b;
  }

  // Sentinels and spacers are loaded when they scroll into view. htmx's own
  // intersect trigger with root:#scroller did not fire after a region swap and
  // an IntersectionObserver only runs while the page is being rendered, so
  // visibility is checked with getBoundingClientRect after every scroll,
  // every swap and on a 100 ms timer.
  function loadVisiblePlaceholders() {
    var s = scroller();
    if (!s) return;
    var r = s.getBoundingClientRect(), hdr = headerHeight(s);
    $$(".sentinel, .sp", s).forEach(function (el) {
      // A trigger sent before htmx processed a fresh node is lost, so a
      // placeholder that is still visible is triggered again after 400 ms
      // (hx-sync drops the duplicate while a request is in flight).
      if (el.dataset.req && Date.now() - +el.dataset.req < 400) return;
      var b = el.getBoundingClientRect();
      if (b.bottom > r.top + hdr && b.top < r.bottom) {
        el.dataset.req = String(Date.now());
        htmx.trigger(el, "gguix-visible");
      }
    });
  }
  G.loadVisible = loadVisiblePlaceholders;
  function watchPlaceholders(s) {
    if (!watchPlaceholders.on) {
      watchPlaceholders.on = true;
      setInterval(loadVisiblePlaceholders, 100);
    }
    s.addEventListener("scroll", function () {
      var h = $("#thead-wrap");
      if (h) h.scrollLeft = s.scrollLeft;
      loadVisiblePlaceholders();
    }, { passive: true });
    loadVisiblePlaceholders();
  }

  // ------------------------------------------------------------ column resize
  // Widths live in the <col> elements of the header table and the body table
  // (table-layout: fixed). They are kept here so a region swap re-applies them.
  var colW = null;
  function applyColWidths() {
    var tables = $$("#thead-wrap table, #scroller table");
    if (!tables.length) return;
    if (!colW) colW = Array.from($$("#thead-wrap col, #scroller col").slice(0, 11), function (c) { return parseFloat(c.style.width); });
    var total = colW.reduce(function (a, b) { return a + b; }, 0);
    tables.forEach(function (t) {
      t.style.width = total + "px";
      var cols = t.querySelectorAll("col");
      for (var i = 0; i < cols.length && i < colW.length; i++) cols[i].style.width = colW[i] + "px";
    });
  }
  document.addEventListener("click", function (e) {
    if (e.target.closest && e.target.closest("[data-col-resize]")) e.stopPropagation();
  }, true);
  document.addEventListener("pointerdown", function (e) {
    var g = e.target.closest && e.target.closest("[data-col-resize]");
    if (!g || e.button !== 0) return;
    e.preventDefault();
    e.stopPropagation();
    applyColWidths();
    var i = +g.dataset.colResize, x0 = e.clientX, w0 = colW[i];
    g.setPointerCapture(e.pointerId);
    g.setAttribute("data-dragging", "");
    function move(ev) {
      colW[i] = Math.max(40, Math.round(w0 + ev.clientX - x0));
      applyColWidths();
    }
    function up() {
      g.removeAttribute("data-dragging");
      g.removeEventListener("pointermove", move);
      g.removeEventListener("pointerup", up);
      g.removeEventListener("pointercancel", up);
    }
    g.addEventListener("pointermove", move);
    g.addEventListener("pointerup", up);
    g.addEventListener("pointercancel", up);
  });
  G.colWidths = function () { applyColWidths(); return colW.slice(); };

  function initScroller() {
    var s = scroller();
    if (!s) return;
    applyColWidths();
    watchPlaceholders(s);
    document.body.dataset.view = s.dataset.view;
    var r = +s.dataset.scrollRow;
    if (r >= 0) G.scrollToRow(r); else s.scrollTop = 0;
    markSelected(selectedId());
    $$(".pg", s).forEach(function (b) { b.dataset.seen = "1"; });
    updateRowsStatus();
    document.dispatchEvent(new CustomEvent("gguix:region"));
  }

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

  // Keep at most MAX_LOADED pages in the DOM: replace the page furthest from
  // the viewport with a spacer of the same height.
  function evict(justLoaded) {
    var s = scroller();
    if (!s) return;
    var blocks = $$(".pg", s);
    while (blocks.length > MAX_LOADED) {
      var r = s.getBoundingClientRect();
      var worst = null, worstD = -1;
      blocks.forEach(function (b) {
        if (b === justLoaded) return;
        var rect = b.getBoundingClientRect();
        var d = rect.bottom < r.top ? r.top - rect.bottom : rect.top > r.bottom ? rect.top - r.bottom : 0;
        if (d > worstD) { worst = b; worstD = d; }
      });
      if (!worst) return;
      toSpacer(worst);
      blocks = $$(".pg", s);
    }
  }

  function toSpacer(block) {
    var h = block.getBoundingClientRect().height;
    var table = block.tagName === "TBODY";
    var sp = document.createElement(table ? "tbody" : "div");
    sp.className = "sp";
    sp.dataset.page = block.dataset.page;
    sp.dataset.src = block.dataset.src;
    sp.setAttribute("data-slot", "placeholder");
    sp.setAttribute("hx-get", block.dataset.src);
    sp.setAttribute("hx-trigger", "gguix-visible once");
    sp.setAttribute("hx-swap", "outerHTML");
    sp.setAttribute("hx-sync", "this:drop");
    if (table) {
      sp.innerHTML = '<tr style="height:' + h + 'px"><td colspan="11" class="text-center text-muted-foreground">…</td></tr>';
    } else {
      sp.style.height = h + "px";
      sp.className = "sp mb-0";
      sp.innerHTML = '<div class="text-center text-muted-foreground">…</div>';
    }
    block.replaceWith(sp);
    htmx.process(sp);
  }

  var anchor = null;
  document.addEventListener("htmx:beforeSwap", function (e) {
    var t = e.detail.target;
    anchor = null;
    if (!t || !t.matches || !t.matches(".sentinel, .sp")) return;
    var s = scroller();
    var next = t.nextElementSibling;
    if (!s || !next) return;
    var up = t.dataset.dir === "up";
    var above = t.getBoundingClientRect().top < s.getBoundingClientRect().top + headerHeight(s);
    if (up || above) anchor = { el: next, top: next.getBoundingClientRect().top };
  });

  document.addEventListener("htmx:afterSwap", function (e) {
    var s = scroller();
    if (!s) return;
    var t = e.detail.target;
    if (anchor && anchor.el.isConnected) {
      var d = anchor.el.getBoundingClientRect().top - anchor.top;
      if (d) s.scrollTop += d;
    }
    anchor = null;
    if (e.detail.requestConfig && /^\/page\?/.test(e.detail.requestConfig.path || "")) {
      var fresh = $$(".pg", s).filter(function (b) { return !b.dataset.seen; });
      fresh.forEach(function (b) { b.dataset.seen = "1"; });
      evict(fresh[0]);
      updateRowsStatus();
    }
    G.loadVisible();
  });

  // In-flight counting for bench "loaded" detection.
  // Counted per XHR and released on loadend, so aborted requests cannot leave
  // the counter stuck.
  document.addEventListener("htmx:beforeSend", function (e) {
    var xhr = e.detail.xhr;
    inflight++;
    var released = false;
    xhr.addEventListener("loadend", function () {
      setTimeout(function () { if (!released) { released = true; inflight = Math.max(0, inflight - 1); } }, 0);
    });
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
  function selectedRowIdx() {
    var s = scroller();
    var el = s && s.querySelector('[data-state="selected"]');
    return el ? el.dataset.row : undefined;
  }
  G.selectedRowIdx = selectedRowIdx;

  function markSelected(id) {
    var s = scroller();
    if (!s) return;
    $$('[data-state="selected"]', s).forEach(function (el) {
      el.removeAttribute("data-state");
      el.removeAttribute("aria-selected");
    });
    if (!id) return;
    $$('[data-id="' + id + '"]', s).forEach(function (el) {
      el.setAttribute("data-state", "selected");
      el.setAttribute("aria-selected", "true");
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

  document.addEventListener("keydown", function (e) {
    var s = scroller();
    if (!s || document.activeElement !== s) return;
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    e.preventDefault();
    var items = $$("[data-id]", s);
    if (!items.length) return;
    var cur = s.querySelector('[data-state="selected"]');
    var i = cur ? items.indexOf(cur) : -1;
    var n = e.key === "ArrowDown" ? Math.min(items.length - 1, i + 1) : Math.max(0, i < 0 ? 0 : i - 1);
    var el = items[n];
    if (!el) return;
    var r = s.getBoundingClientRect(), hdr = headerHeight(s), b = el.getBoundingClientRect();
    if (b.top < r.top + hdr) s.scrollTop -= r.top + hdr - b.top;
    else if (b.bottom > r.bottom) s.scrollTop += b.bottom - r.bottom;
    selectItem(parseInt(el.dataset.id, 10));
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

  document.addEventListener("htmx:confirm", function (e) {
    var el = e.detail.elt;
    if (el && el.id === "goto" && isDirty()) {
      e.preventDefault();
      guard(function () { e.detail.issueRequest(true); });
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

  G.gotoRow = function (n) {
    var done = whenRegion();
    htmx.ajax("POST", "/goto", { values: { row: n }, target: "#list-region", swap: "innerHTML" });
    return done;
  };

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
    // The native menu bar owns these shortcuts as key equivalents.
    if (document.body.dataset.nativeMenu === "true") return;
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
