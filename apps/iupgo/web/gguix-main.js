// Main-thread helpers. The IUP wasm loader runs the Go app in a Worker, which
// has no document, no file input and no clipboard API, so the app posts
// requests here: browser download, notes file picker, clipboard copy.
window.gguixAttach = function (worker) {
  worker.addEventListener('message', function (e) {
    var d = e.data;
    if (!d || typeof d !== 'object') return;
    if (d.__gguixDownload) {
      var url = URL.createObjectURL(new Blob([d.data], { type: 'text/csv' }));
      var a = document.createElement('a');
      a.href = url; a.download = d.name;
      document.body.appendChild(a); a.click(); document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } else if (d.__gguixCopy) {
      if (navigator.clipboard) navigator.clipboard.writeText(d.text).catch(function () {});
    } else if (d.__gguixPick) {
      var inp = document.createElement('input');
      inp.type = 'file'; inp.accept = '.txt,text/plain';
      inp.addEventListener('change', function () {
        var f = inp.files[0];
        if (!f) { worker.postMessage({ __gguixPicked: true, cancel: true }); return; }
        if (f.size > 64 * 1024) { worker.postMessage({ __gguixPicked: true, tooBig: true }); return; }
        f.text().then(function (t) { worker.postMessage({ __gguixPicked: true, text: t }); });
      });
      inp.addEventListener('cancel', function () { worker.postMessage({ __gguixPicked: true, cancel: true }); });
      inp.click();
    }
  });
};
