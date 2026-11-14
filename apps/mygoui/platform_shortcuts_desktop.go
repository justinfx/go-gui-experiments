//go:build !js

package main

import "github.com/egoist/mygo/ui"

// The desktop has the native menu bar, whose accelerators do this.
func (a *app) platformShortcuts(c *ui.Context) {}
