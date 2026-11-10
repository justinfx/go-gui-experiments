//go:build !js

package main

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/test"

	"gguix/core/host"
	"gguix/core/memory"
)

// expectedHost is what the Host panel must show on the desktop target.
func expectedHost() host.Info { return host.Current(appID) }

func notFeasibleScenarios() []string { return nil }

func testExportCancel(t *testing.T) {
	h := newHarness(t, memory.Options{Rows: 1000, Latency: 0, ExportRowsPerSec: 200})
	h.loaded()
	path := filepath.Join(t.TempDir(), "gguix-export.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	h.u.exportTo(f, path)
	if !h.u.exporting || !h.u.exportBtn.Disabled() || !h.u.exportBox.Visible() {
		t.Fatal("export must show progress and disable the Export button")
	}
	h.wait("some progress", func() bool { return h.u.progress.Value > 0.02 })
	test.Tap(h.u.cancelBtn)
	h.wait("export to stop", func() bool { return !h.u.exporting })
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("partial file must be removed, stat err = %v", err)
	}
	if h.u.exportBtn.Disabled() {
		t.Error("Export button must be enabled again")
	}
	if h.u.flashLbl.Text != "Export cancelled" {
		t.Errorf("status message = %q, want Export cancelled", h.u.flashLbl.Text)
	}
}
