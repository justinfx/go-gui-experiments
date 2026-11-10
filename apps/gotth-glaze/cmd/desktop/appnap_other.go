//go:build !darwin

package main

// disableAppNap is only needed on macOS.
func disableAppNap() {}
