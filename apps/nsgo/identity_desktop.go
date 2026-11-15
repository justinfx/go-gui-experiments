//go:build linux

package main

import "github.com/ebitengine/purego"

// On Linux NSGo opens its GTK window without a program name, so X11 shows an
// empty WM_CLASS and Wayland the fallback app id "GTK Application" (nsgo.yaml's
// id only reaches core.App, which NSGo uses for notifications). GTK takes
// both from g_get_prgname(), so the app sets it, with purego (a dependency of
// NSGo already), before NSGo creates the window. Failure is harmless.
func init() {
	glib, err := purego.Dlopen("libglib-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	var setPrgname, setAppName func(string)
	purego.RegisterLibFunc(&setPrgname, glib, "g_set_prgname")
	purego.RegisterLibFunc(&setAppName, glib, "g_set_application_name")
	setPrgname("dev.gguix.nsgo")
	setAppName("GGUIX - NSGo")
}
