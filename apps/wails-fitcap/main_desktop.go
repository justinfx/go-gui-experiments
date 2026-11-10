//go:build !server

package main

// Desktop entry points (category: desktop): the window, the native menu,
// the notification service and the quit sequence.

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// notifier is only registered inside an app bundle: the macOS notification
// center refuses to start without a bundle identifier, and a failing service
// stops the whole app.
var notifier *notifications.NotificationService

func inBundle() bool {
	exe, err := os.Executable()
	return err == nil && strings.Contains(exe, ".app/Contents/MacOS/")
}

func targetServices() []application.Service {
	if runtime.GOOS == "darwin" && !inBundle() {
		return nil
	}
	notifier = notifications.New()
	return []application.Service{application.NewService(notifier)}
}

func assetMiddleware(*ItemService) application.Middleware { return nil }

// quitting is set once the frontend has flushed its preferences.
var quitting atomic.Bool

func configureTarget(o *application.Options, svc *ItemService) {
	o.ShouldQuit = func() bool {
		if quitting.Load() {
			return true
		}
		// Ask the frontend to save its pending preferences first, then quit.
		emitEvent("app:flush", nil)
		go func() {
			select {
			case <-svc.flushed:
			case <-time.After(400 * time.Millisecond):
			}
			quitting.Store(true)
			application.Get().Quit()
		}()
		return false
	}
	o.OnShutdown = func() { svc.CancelExport() }
}

func setupTarget(app *application.App, svc *ItemService) {
	p, _ := svc.store.Load()
	w, h := p.WindowWidth, p.WindowHeight
	if w <= 0 {
		w = 1280
	}
	if h <= 0 {
		h = 800
	}
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "GGUIX - Wails",
		Width:     w,
		Height:    h,
		MinWidth:  1024,
		MinHeight: 640,
		URL:       "/",
	})
	svc.plat = &desktopPlatform{app: app, window: win, notifier: notifier}

	var timer *time.Timer
	var tmu sync.Mutex
	win.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) {
		tmu.Lock()
		defer tmu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(400*time.Millisecond, func() {
			cw, ch := win.Size()
			svc.noteWindowSize(cw, ch)
		})
	})

	app.Menu.Set(buildMenu(app, svc))
	svc.onMenuState = menuStateUpdater()
}

var _ = runtime.GOOS
