// Desktop shell glue: the webview has no save/open dialogs, notifications or
// reliable clipboard, so these go to Go (see cmd/desktop and handlers_desktop.go).
(function () {
  "use strict";
  var G = (window.GGUIX = window.GGUIX || {});
  var form = function (obj) { return new URLSearchParams(obj).toString(); };
  var post = function (url, body) {
    return fetch(url, { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: body || "" });
  };

  G.platform = {
    startExport: function () {
      return post("/export/save").then(function (r) {
        if (!r.ok && r.status !== 204 && r.status !== 202) return r.text().then(function (t) { G.showError(t); });
      });
    },
    copyText: function (text) {
      return post("/native/copy", form({ text: text })).then(function () {
        if (navigator.clipboard && navigator.clipboard.writeText) navigator.clipboard.writeText(text).catch(function () {});
      });
    },
    importNotes: function () {
      return post("/native/open-notes").then(function (r) { return r.json(); }).then(function (j) {
        if (j.error) return { error: j.error };
        if (!j.ok) return null;
        return { text: j.text };
      });
    },
    // The server sends the OS notification when the export finishes.
    notify: function () {}
  };

  // Push view, theme and export state to the native menu bar (a no-op when the
  // page draws its own menu). gguixMenuState is bound by cmd/desktop.
  var hasExport = false;
  function syncMenu() {
    if (typeof window.gguixMenuState !== "function") return;
    window.gguixMenuState(document.body.dataset.view || "table", document.documentElement.dataset.theme || "system",
      G.exportState && G.exportState() === "running", hasExport);
  }
  document.addEventListener("gguix:export", function (e) {
    if (e.detail.state === "done") hasExport = true;
    syncMenu();
  });
  document.addEventListener("DOMContentLoaded", function () {
    syncMenu();
    var mo = new MutationObserver(syncMenu);
    mo.observe(document.body, { attributes: true, attributeFilter: ["data-view"] });
    mo.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
  });

  // Edit menu entries drive the focused text control.
  G.actions["cut"] = function () { document.execCommand("cut"); };
  G.actions["copy"] = function () { document.execCommand("copy"); };
  G.actions["select-all"] = function () { document.execCommand("selectAll"); };
  G.actions["paste"] = function () {
    if (navigator.clipboard && navigator.clipboard.readText) {
      navigator.clipboard.readText().then(function (t) { document.execCommand("insertText", false, t); }).catch(function () {});
    }
  };

  // Reveal Last Export is disabled until an export finishes.
  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll('[data-menu-action="reveal-export"]').forEach(function (it) {
      it.setAttribute("data-disabled", "");
      it.setAttribute("aria-disabled", "true");
    });
  });

  // Report the window size so the next launch can restore it.
  var t;
  window.addEventListener("resize", function () {
    clearTimeout(t);
    t = setTimeout(function () { post("/prefs/window", form({ w: window.innerWidth, h: window.innerHeight })); }, 500);
  });
})();
