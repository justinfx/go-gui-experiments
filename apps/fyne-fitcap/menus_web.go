//go:build js && wasm

package main

// The web targets have no menu bar.

type menuItems struct{}

func (u *ui) updateMenus() {}
