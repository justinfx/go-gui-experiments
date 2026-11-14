// Test-only stand-in for window.mygo, which MyGo injects into its own
// webview windows. The e2e harness (main_e2e.go, tag e2e) serves the Go
// service over HTTP; this forwards the generated client's calls to it.
(() => {
  const call = async (method, ...args) => {
    const r = await fetch("/rpc/" + method, { method: "POST", body: JSON.stringify(args) });
    if (!r.ok) {
      const e = new Error((await r.text()).trim());
      e.name = "CallError";
      e.method = method;
      throw e;
    }
    return r.json();
  };
  let es;
  const listeners = {};
  const on = (name, fn) => {
    es ??= new EventSource("/events");
    const h = (m) => fn(JSON.parse(m.data));
    es.addEventListener(name, h);
    return () => es.removeEventListener(name, h);
  };
  window.mygo = {
    call,
    on,
    once: (n, f) => {
      const off = on(n, (p) => (off(), f(p)));
      return off;
    },
    platform: "darwin",
    windowId: 1,
    version: "e2e",
    preferences: {},
    locale: "en",
    dir: "ltr",
    windowActive: true,
  };
  void listeners;
})();
