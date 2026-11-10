//go:build js && !remote

package host

// Current returns Source "unavailable": the browser sandbox has no host access.
func Current(appID string) Info { return unavailable(appID) }

// Server returns Source "unavailable" in js/wasm builds.
func Server(appID string) Info { return unavailable(appID) }
