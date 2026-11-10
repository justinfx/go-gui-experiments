// Web target glue: browser features (download, Clipboard API, Notification
// API, file input). The desktop shell uses static/desktop/desktop.js instead.
(function () {
  "use strict";
  var G = (window.GGUIX = window.GGUIX || {});

  G.platform = {
    // Export: the server streams the CSV; the browser saves it as a download.
    startExport: function () {
      if ("Notification" in window && Notification.permission === "default") {
        try { Notification.requestPermission(); } catch (e) { /* ignore */ }
      }
      if (window.GGUIX_BENCH) {
        // Bench: no download, the CSV is read and discarded.
        return fetch("/export").then(function (r) {
          var rd = r.body.getReader();
          return (function pump() { return rd.read().then(function (x) { if (!x.done) return pump(); }); })();
        });
      }
      var a = document.createElement("a");
      a.href = "/export";
      a.download = "gguix-export.csv";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
    },

    copyText: function (text) {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        return navigator.clipboard.writeText(text).catch(function () { return legacyCopy(text); });
      }
      return legacyCopy(text);
    },

    // Import Notes: a file input, filtered to .txt.
    importNotes: function () {
      return new Promise(function (resolve) {
        var inp = document.createElement("input");
        inp.type = "file";
        inp.accept = ".txt,text/plain";
        inp.addEventListener("change", function () {
          var f = inp.files && inp.files[0];
          if (!f) return resolve(null);
          if (f.size > 64 * 1024) return resolve({ error: "File is too large." });
          f.text().then(function (t) { resolve({ text: t }); }, function (e) { resolve({ error: String(e) }); });
        });
        inp.addEventListener("cancel", function () { resolve(null); });
        inp.click();
      });
    },

    // Notification API, with the in-app toast as the fallback.
    notify: function (title, body) {
      if ("Notification" in window && Notification.permission === "granted") {
        try { new Notification(title, { body: body }); return; } catch (e) { /* fall through */ }
      }
      if (window.templ && window.templ.toast) {
        window.templ.toast.add({ title: title, description: body, type: "success" });
      } else {
        G.flash(title + ": " + body, 4000);
      }
    }
  };

  function legacyCopy(text) {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand("copy"); } catch (e) { /* ignore */ }
    document.body.removeChild(ta);
  }
})();
