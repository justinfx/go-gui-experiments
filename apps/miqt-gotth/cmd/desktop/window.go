package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/webchannel"
	"github.com/mappu/miqt/qt6/webengine"
)

// qwebchannel.js is the client half of QWebChannel, copied from the Qt
// install (share/qt6/webchannel/qwebchannel.js). A page loaded over http
// cannot reach the qrc: copy, so it is injected as a user script.
//
//go:embed qwebchannel.js
var qwebchannelJS string

// bridgeJS runs in the page's main world before the page's own scripts. It
// defines window.gguixMenuState, the function static/desktop.js calls to push
// view, theme and export state to the menu bar. The call travels over
// QWebChannel by writing the "text" property of a registered QLineEdit; MIQT
// has no way to declare a Go slot that JavaScript can invoke (that needs moc),
// so a QObject property that Go observes is the only Go bound route.
const bridgeJS = `
(function () {
  var obj = null, pending = null, seq = 0;
  function flush() {
    if (obj && pending) { obj.text = (++seq) + ":" + JSON.stringify(pending); pending = null; }
  }
  window.gguixMenuState = function (view, theme, exporting, hasExport) {
    pending = { view: view, theme: theme, exporting: !!exporting, hasExport: !!hasExport };
    flush();
  };
  function init() {
    if (!window.qt || !qt.webChannelTransport) { setTimeout(init, 5); return; }
    new QWebChannel(qt.webChannelTransport, function (ch) { obj = ch.objects.bridge; flush(); });
  }
  init();
})();
`

type window struct {
	*qt.QMainWindow
	view *webengine.QWebEngineView

	exportAct, revealAct *qt.QAction
	viewActs             map[string]*qt.QAction
	themeActs            map[string]*qt.QAction
	bridge               *qt.QLineEdit
}

func newWindow(w, h int) *window {
	win := &window{
		QMainWindow: qt.NewQMainWindow2(),
		viewActs:    map[string]*qt.QAction{},
		themeActs:   map[string]*qt.QAction{},
	}
	win.SetWindowTitle("GGUIX - GoTTH")
	win.SetMinimumSize2(1024, 640)
	win.Resize(w, h)

	win.view = webengine.NewQWebEngineView2()
	win.SetCentralWidget(win.view.QWidget)

	// QWebChannel with one registered object (see bridgeJS).
	win.bridge = qt.NewQLineEdit2()
	win.bridge.OnTextChanged(win.onBridge)
	ch := webchannel.NewQWebChannel()
	ch.RegisterObject("bridge", win.bridge.QObject)
	win.view.Page().SetWebChannel(ch)
	sc := webengine.NewQWebEngineScript()
	sc.SetName("gguix-bridge")
	sc.SetSourceCode(qwebchannelJS + "\n" + bridgeJS)
	sc.SetInjectionPoint(webengine.QWebEngineScript__DocumentCreation)
	sc.SetWorldId(uint(webengine.QWebEngineScript__MainWorld))
	win.view.Page().Scripts().Insert(sc)

	win.buildMenus()
	return win
}

// onBridge receives "<seq>:<json>" from the page and updates the menu state.
func (w *window) onBridge(s string) {
	i := strings.IndexByte(s, ':')
	if i < 0 {
		return
	}
	var st struct {
		View, Theme          string
		Exporting, HasExport bool
	}
	if json.Unmarshal([]byte(s[i+1:]), &st) != nil {
		return
	}
	if os.Getenv("GOTTH_LOG") == "1" {
		log.Printf("bridge: %s", s[i+1:])
	}
	for k, a := range w.viewActs {
		a.SetChecked(k == st.View)
	}
	for k, a := range w.themeActs {
		a.SetChecked(k == st.Theme)
	}
	w.exportAct.SetEnabled(!st.Exporting)
	w.revealAct.SetEnabled(st.HasExport)
}

// action runs the same GGUIX.actions[name] entry the in-page menu and the
// keyboard use (static/app.js). Names are fixed ASCII constants.
func (w *window) action(name string) {
	w.view.Page().RunJavaScriptWithScriptSource(
		"(function(){var f=window.GGUIX&&GGUIX.actions&&GGUIX.actions[" + strconv.Quote(name) + "];if(f)f();})()")
}

func (w *window) buildMenus() {
	mb := w.MenuBar()
	add := func(m *qt.QMenu, text, shortcut, act string) *qt.QAction {
		a := qt.NewQAction5(text, m.QObject)
		if shortcut != "" {
			a.SetShortcut(qt.NewQKeySequence2(shortcut))
		}
		a.OnTriggered(func() { w.action(act) })
		m.QWidget.AddAction(a)
		return a
	}
	page := func(m *qt.QMenu, text, shortcut string, wa webengine.QWebEnginePage__WebAction) {
		a := qt.NewQAction5(text, m.QObject)
		a.SetShortcut(qt.NewQKeySequence2(shortcut))
		a.OnTriggered(func() { w.view.TriggerPageAction(wa) })
		m.QWidget.AddAction(a)
	}

	// Qt moves role-tagged actions to the application menu on macOS.
	app := mb.AddMenuWithTitle("GGUIX")
	add(app, "About GGUIX", "", "about").SetMenuRole(qt.QAction__AboutRole)
	add(app, "Quit GGUIX", "Ctrl+Q", "quit").SetMenuRole(qt.QAction__QuitRole)

	file := mb.AddMenuWithTitle("File")
	w.exportAct = add(file, "Export CSV…", "Ctrl+E", "export")
	add(file, "Import Notes…", "Ctrl+O", "import-notes")
	w.revealAct = add(file, "Reveal Last Export", "", "reveal-export")
	w.revealAct.SetEnabled(false)

	edit := mb.AddMenuWithTitle("Edit")
	page(edit, "Cut", "Ctrl+X", webengine.QWebEnginePage__Cut)
	page(edit, "Copy", "Ctrl+C", webengine.QWebEnginePage__Copy)
	page(edit, "Paste", "Ctrl+V", webengine.QWebEnginePage__Paste)
	page(edit, "Select All", "Ctrl+A", webengine.QWebEnginePage__SelectAll)
	edit.AddSeparator()
	add(edit, "Copy Row", "Ctrl+Shift+C", "copy-row")

	view := mb.AddMenuWithTitle("View")
	vg := qt.NewQActionGroup(view.QObject)
	for _, v := range []struct{ key, text, sc string }{{"table", "Table", "Ctrl+1"}, {"cards", "Cards", "Ctrl+2"}} {
		a := add(view, v.text, v.sc, "view-"+v.key)
		a.SetCheckable(true)
		vg.AddAction(a)
		w.viewActs[v.key] = a
	}
	w.viewActs["table"].SetChecked(true)
	view.AddSeparator()
	theme := view.AddMenuWithTitle("Theme")
	tg := qt.NewQActionGroup(theme.QObject)
	for _, t := range []struct{ key, text string }{{"system", "System"}, {"light", "Light"}, {"dark", "Dark"}} {
		a := add(theme, t.text, "", "theme-"+t.key)
		a.SetCheckable(true)
		tg.AddAction(a)
		w.themeActs[t.key] = a
	}
	w.themeActs["system"].SetChecked(true)
	view.AddSeparator()
	add(view, "Reload", "Ctrl+R", "reload")
	add(view, "Focus Search", "Ctrl+F", "focus-search")

	help := mb.AddMenuWithTitle("Help")
	add(help, "Host Info…", "Ctrl+I", "host")
}
