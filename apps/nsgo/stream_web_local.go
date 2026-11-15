//go:build js && !remote

package main

// streamHook is nil for the in-process core: the whole export is built in
// memory and handed to SaveFile as one download (section 5.5).
func streamHook() func(string, func(*streamSink, bool, error)) { return nil }
