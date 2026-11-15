//go:build matrix && !js

package main

import "github.com/gen2brain/iup-go/iup"

// IupMatrixEx lives in the controls library, opened after IupOpen.
func openControls() { iup.ControlsOpen() }
