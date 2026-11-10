//go:build darwin && native

package main

/*
#cgo CFLAGS: -fobjc-arc -mmacosx-version-min=11.0
#cgo LDFLAGS: -framework Cocoa -framework UserNotifications -framework UniformTypeIdentifiers
#include <stdlib.h>
void gguixInstallMenu(void *window);
void gguixMenuState(const char *view, const char *theme, int exporting, int hasExport);
char *gguixSaveDialog(const char *defaultName);
char *gguixOpenDialog(const char *ext);
int gguixNotify(const char *title, const char *body);
void gguixCopyText(const char *text);
*/
import "C"

import (
	"errors"
	"unsafe"

	webview "github.com/webview/webview_go"
)

// shimNative implements gotth.Native with the Objective-C shim in
// native_darwin.m: NSOpenPanel / NSSavePanel sheets on the webview's NSWindow,
// a user notification, and NSPasteboard. The webview library has no API for
// any of these, only the NSWindow pointer from Window().
type shimNative struct{}

func (shimNative) SaveDialog(defaultName string) (string, bool, error) {
	n := C.CString(defaultName)
	defer C.free(unsafe.Pointer(n))
	return takePath(C.gguixSaveDialog(n))
}

func (shimNative) OpenDialog(ext string) (string, bool, error) {
	e := C.CString(ext)
	defer C.free(unsafe.Pointer(e))
	return takePath(C.gguixOpenDialog(e))
}

func takePath(p *C.char) (string, bool, error) {
	if p == nil {
		return "", false, nil // cancelled
	}
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p), true, nil
}

func (shimNative) Notify(title, body string) error {
	t, b := C.CString(title), C.CString(body)
	defer C.free(unsafe.Pointer(t))
	defer C.free(unsafe.Pointer(b))
	if C.gguixNotify(t, b) != 0 {
		return errors.New("notification could not be delivered")
	}
	return nil
}

func (shimNative) CopyText(text string) error {
	t := C.CString(text)
	defer C.free(unsafe.Pointer(t))
	C.gguixCopyText(t)
	return nil
}

var shimWindow webview.WebView

//export gguixGoMenuAction
func gguixGoMenuAction(name *C.char) {
	// The same actions the in-page menu and the keyboard use (static/app.js).
	js := "(function(){var f=window.GGUIX&&GGUIX.actions&&GGUIX.actions[" + quoteJS(C.GoString(name)) + "];if(f)f();})()"
	shimWindow.Eval(js) // called on the main thread by AppKit
}

func quoteJS(s string) string { return `"` + s + `"` } // names are fixed ASCII constants

// installNativeMenu installs the NSMenu main menu on the process and lets the
// page push radio/enabled state to it. It returns true when installed.
func installNativeMenu(w webview.WebView) bool {
	shimWindow = w
	C.gguixInstallMenu(w.Window())
	_ = w.Bind("gguixMenuState", func(view, theme string, exporting, hasExport bool) {
		v, t := C.CString(view), C.CString(theme)
		defer C.free(unsafe.Pointer(v))
		defer C.free(unsafe.Pointer(t))
		var ex, he C.int
		if exporting {
			ex = 1
		}
		if hasExport {
			he = 1
		}
		C.gguixMenuState(v, t, ex, he)
	})
	return true
}

// platformNative is the native bridge of the `native` build.
func platformNative() gotthNative { return shimNative{} }
