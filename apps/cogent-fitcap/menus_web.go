//go:build js

package main

import "cogentcore.org/core/core"

// The web targets have no menu bar (brief section 5.6 is desktop only).
func (a *App) addMenuBar(b *core.Body) {}
func (a *App) refreshMenus()           {}
