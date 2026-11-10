//go:build !(darwin && native)

package main

import (
	"errors"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
	"github.com/ncruces/zenity"

	webview "github.com/webview/webview_go"
)

// osaNative implements gotth.Native with small cross-platform, cgo-free
// libraries (the webview library has no API for any of these):
//   - zenity for open/save dialogs (macOS: osascript; Windows: native
//     common dialogs; Linux: the zenity/kdialog/matedialog binaries)
//   - beeep for notifications (macOS: osascript; Windows: toast;
//     Linux: org.freedesktop.Notifications over D-Bus)
//   - atotto/clipboard for copy (macOS: pbcopy; Windows: Win32;
//     Linux: xclip, xsel or wl-copy)
//
// The type name is kept from the first, macOS-only version.
type osaNative struct{}

func (osaNative) SaveDialog(defaultName string) (string, bool, error) {
	p, err := zenity.SelectFileSave(zenity.Title("Export CSV"), zenity.Filename(defaultName), zenity.ConfirmOverwrite())
	if errors.Is(err, zenity.ErrCanceled) {
		return "", false, nil
	}
	return p, err == nil, err
}

func (osaNative) OpenDialog(ext string) (string, bool, error) {
	pat := "*." + strings.TrimPrefix(ext, ".")
	p, err := zenity.SelectFile(zenity.Title("Import Notes"), zenity.FileFilter{Name: pat, Patterns: []string{pat}, CaseFold: true})
	if errors.Is(err, zenity.ErrCanceled) {
		return "", false, nil
	}
	return p, err == nil, err
}

func (osaNative) Notify(title, body string) error {
	return beeep.Notify(title, body, "")
}

func (osaNative) CopyText(text string) error {
	return clipboard.WriteAll(text)
}

func platformNative() gotthNative { return osaNative{} }

// installNativeMenu reports false: the page draws its own menu bar.
func installNativeMenu(webview.WebView) bool { return false }
