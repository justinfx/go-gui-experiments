// The IUP wasm loader needs SharedArrayBuffer, which needs cross-origin
// isolation. The core server does not send the COOP/COEP headers, so a
// service worker adds them (registered here, then the page reloads once).
(function () {
  if (window.crossOriginIsolated || !('serviceWorker' in navigator)) return;
  navigator.serviceWorker.register('coi-sw.js').then(function (reg) {
    var go = function () { window.location.reload(); };
    if (reg.active) { go(); return; }
    var sw = reg.installing || reg.waiting;
    if (sw) sw.addEventListener('statechange', function () { if (sw.state === 'activated') go(); });
  });
  throw new Error('waiting for cross-origin isolation');
})();
