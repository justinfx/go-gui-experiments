package main

import (
	"errors"
	"sync"

	"github.com/crgimenes/glaze"
	"github.com/crgimenes/native/clipboard"
	"github.com/crgimenes/native/notify"
	"github.com/crgimenes/native/openurl"
)

// glazeNative implements gotth.Native with glaze's own file dialogs and the
// crgimenes/native notify, clipboard and openurl packages. All of them are
// pure Go (purego), so the desktop binary builds with CGO_ENABLED=0.
type glazeNative struct {
	mu sync.Mutex
	w  glaze.WebView
}

func newNative(w glaze.WebView) *glazeNative { return &glazeNative{w: w} }

// The dialogs block the calling goroutine and must not run on the UI thread;
// the HTTP handlers that call them run on their own goroutines.
func (n *glazeNative) SaveDialog(defaultName string) (string, bool, error) {
	p, err := n.w.SaveFile(glaze.FileDialogOptions{
		Title:    "Export CSV",
		Filename: defaultName,
		Filters:  []glaze.FileFilter{{Name: "CSV", Extensions: []string{"csv"}}},
	})
	return p, p != "", err
}

func (n *glazeNative) OpenDialog(ext string) (string, bool, error) {
	p, err := n.w.OpenFile(glaze.FileDialogOptions{
		Title:   "Import Notes",
		Filters: []glaze.FileFilter{{Name: "Text", Extensions: []string{ext}}},
	})
	return p, p != "", err
}

func (n *glazeNative) Notify(title, body string) error {
	err := notify.Send(notify.Message{Title: title, Body: body})
	if errors.Is(err, errors.ErrUnsupported) {
		return nil // no notification backend on this OS: the in-page toast is the fallback
	}
	return err
}

func (n *glazeNative) CopyText(text string) error { return clipboard.WriteText(text) }

// Reveal shows path in the file manager (optional hook, see handlers_desktop.go).
func (n *glazeNative) Reveal(path string) error { return openurl.Reveal(path) }
