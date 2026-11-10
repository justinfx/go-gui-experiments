//go:build !js

package main

import "cogentcore.org/core/system"

func quitApp() { system.TheApp.Quit() }
