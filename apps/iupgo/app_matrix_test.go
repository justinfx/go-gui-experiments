//go:build matrix && !js

package main

import (
	"testing"

	"gguix/core"
)

// nativeCellChecks: the Matrix variant draws the cells itself (no images), so the spec checks are all there is.
func nativeCellChecks(t *testing.T, it core.Item) {}
