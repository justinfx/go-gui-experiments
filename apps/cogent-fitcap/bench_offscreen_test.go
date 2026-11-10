package main

import (
	"context"
	"os"
	"testing"
	"time"

	"cogentcore.org/core/core"

	"gguix/core/bench"
	"gguix/core/memory"
	"gguix/core/prefs"
)

// TestBenchOffscreen runs the bench script in the framework's offscreen
// window. It is opt-in (GGUIX_BENCH_OFFSCREEN=1) and not part of the scenarios.
func TestBenchOffscreen(t *testing.T) {
	if os.Getenv("GGUIX_BENCH_OFFSCREEN") != "1" {
		t.Skip("opt-in")
	}
	t.Setenv("GGUIX_PREFS_DIR", t.TempDir())
	svc := memory.New(memory.DefaultOptions())
	a := newApp(svc, prefs.NewMemoryStore(), true)
	b := core.NewBody(windowTitle)
	a.buildUI(b)
	b.RenderImage(func() {
		a.async(a.start)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		bench.Run(ctx, svc, benchDriver{a})
	})
}
