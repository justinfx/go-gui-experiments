package main

import (
	"errors"
	"sync"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
)

// qtNative implements gotth.Native with Qt. The server calls it from HTTP
// handler goroutines, so every Qt call is marshalled to the GUI thread with
// mainthread.Wait; the dialogs block only the calling handler.
type qtNative struct {
	win *window // set once the window exists

	mu   sync.Mutex
	tray *qt.QSystemTrayIcon
}

func newQtNative() *qtNative { return &qtNative{} }

func (n *qtNative) parent() *qt.QWidget {
	if n.win == nil {
		return nil
	}
	return n.win.QWidget
}

func (n *qtNative) SaveDialog(defaultName string) (string, bool, error) {
	var p string
	mainthread.Wait(func() {
		p = qt.QFileDialog_GetSaveFileName4(n.parent(), "Export CSV", defaultName, "CSV files (*.csv)")
	})
	return p, p != "", nil
}

func (n *qtNative) OpenDialog(ext string) (string, bool, error) {
	var p string
	mainthread.Wait(func() {
		p = qt.QFileDialog_GetOpenFileName4(n.parent(), "Import Notes", "", "Text files (*."+ext+")")
	})
	return p, p != "", nil
}

func (n *qtNative) CopyText(text string) error {
	mainthread.Wait(func() { qt.QGuiApplication_Clipboard().SetText(text) })
	return nil
}

// Notify shows a QSystemTrayIcon message. The icon is created on first use and
// hidden again a few seconds later, so no permanent status item is left.
func (n *qtNative) Notify(title, body string) error {
	var err error
	mainthread.Wait(func() {
		if !qt.QSystemTrayIcon_IsSystemTrayAvailable() || !qt.QSystemTrayIcon_SupportsMessages() {
			err = errors.New("system tray notifications are not available")
			return
		}
		n.mu.Lock()
		defer n.mu.Unlock()
		if n.tray == nil {
			pm := qt.NewQPixmap2(32, 32)
			pm.FillWithFillColor(qt.NewQColor3(0x25, 0x63, 0xeb))
			n.tray = qt.NewQSystemTrayIcon2(qt.NewQIcon2(pm))
		}
		n.tray.Show()
		n.tray.ShowMessage3(title, body, qt.NewQIcon(), 5000)
		time.AfterFunc(8*time.Second, func() { mainthread.Start(func() { n.tray.Hide() }) })
	})
	return err
}
