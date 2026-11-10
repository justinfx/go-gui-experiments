# Errata for implementer and judge notes

The per-app notes (`results/<id>/*.md`, `results/<id>-fit/*.md`) and `judge/scores.md` are kept as written. The corrections below come from a fact-check of the exact module sources in the Go module cache, with each item re-checked by hand. REPORT.md uses the corrected facts.

| Note | Original claim | Correction | Evidence |
|---|---|---|---|
| wails friction.md:29-32, toolchain.md:16, progress.md:21 (and wails-fit) | @tanstack/svelte-virtual peer dependency is Svelte ^4 | Its peer range is `^3.48.0 \|\| ^4.0.0 \|\| ^5.0.0`; ^4.2.20 is a devDependency. It is store-based, so a wrapper is a style choice | node_modules/@tanstack/svelte-virtual/package.json |
| wails friction.md:89, progress.md:21 | No reveal-in-Finder API | `app.Env.OpenFileManager(path, true)` exists in v3.1.0 | wails v3 pkg/application |
| wails capabilities.md:15 | shadcn-svelte has no resizable component | Not in the brief's list, but shadcn-svelte ships Resizable (PaneForge) | shadcn-svelte docs |
| miqt friction.md:64 (and miqt-fit) | The only Qt 6 static Windows image has Go 1.26 | Every Qt 6 Windows image in miqt v0.15.0 has Go 1.26 or older (newest win64-cross-go1.26-qt6.5-static); the only Go 1.27 Windows images are Qt 5.15 | miqt@v0.15.0/docker/ |
| miqt (several), judge scores.md:3 | Desktop only | No browser target, but Android ARM64 is supported and Qt WebEngine/WebChannel are bound | miqt README "Supported platforms", qt6/webengine |
| fyne capabilities.md:8, judge scores.md:47,114 | No min window size | No min-size call, but the glfw driver sets size limits from the content's MinSize | internal/driver/glfw/window_desktop.go:300 |
| fyne friction.md:107 (and judge) | No native file dialogs | Fyne-drawn on macOS and Windows; native portal for Linux flatpak builds, OS picker on mobile | dialog/file_xdg_flatpak.go, file_mobile.go |
| fyne friction.md:87 | fyne-cross copies only the app directory | It bind-mounts the app root (or `-dir`) as /app; setting `-dir` to a common parent is an untried alternative to vendoring | fyne-cross internal/command/flag.go:133 |
| fyne capabilities.md:46 | macOS notifications use osascript | Only for unbundled binaries | app/app_darwin.go |
| cogent capabilities.md:9, judge scores.md:74,121 | No min window size | No portable API; reachable via exported `desktop.Window.Glw` glfw handle (untried) | system/driver/desktop/window.go:27-31 |
| cogent judge scores.md:122 | Cross compile unsupported | The `core` CLI route is unsupported (documented); Docker builds work | docs/content/build.md:93 |
| gotth capabilities.md:50,72,121, judge scores.md:101 | webview_go has no menus or dialogs | No API for them, but `Window()` exposes the native window, so cgo can add them | webview_go webview.go:83,173 |
| gotth friction.md:24 | no `datepicker` item | `date-picker` exists as a composition item with no files of its own; `datepicker` and `sonner` are not valid names | shadcn-templ v2.0.0 registry.json |
| gotth capabilities.md:49 | WKWebView has no Notification API | Not tested on macOS WKWebView; public sources cover iOS only. Treat as unverified | WebKit bug 247315 |
