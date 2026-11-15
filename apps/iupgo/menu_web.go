//go:build js

package main

// buildMenu: the web target has no menu bar (the brief's menus are desktop
// only); the toolbar and form buttons cover every action.
func (a *app) buildMenu() {}
