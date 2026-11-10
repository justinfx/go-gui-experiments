package prefs_test

import (
	"os"
	"strings"
	"testing"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/prefs"
)

func TestFileStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(prefs.EnvDir, dir)
	s := prefs.NewFileStore("demo")
	p, err := s.Load()
	if err != nil || p.Theme != core.ThemeSystem || p.View != core.ViewTable {
		t.Fatalf("defaults: %v %+v", err, p)
	}
	p.Theme = core.ThemeDark
	p.Categories = []core.Category{core.Toys}
	p.Status = core.StatusPtr(core.Discontinued)
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(s.Path())
	if !strings.Contains(string(b), `"theme":"dark"`) || !strings.Contains(string(b), `"categories":["Toys"]`) {
		t.Fatalf("file: %s", b)
	}
	back, err := prefs.NewFileStore("demo").Load()
	if err != nil || back.Theme != core.ThemeDark || *back.Status != core.Discontinued {
		t.Fatalf("%v %+v", err, back)
	}
}

func TestHostCurrent(t *testing.T) {
	t.Setenv(prefs.EnvDir, "/tmp/x")
	i := host.Current("demo")
	if i.Source != host.SourceNative || i.Username == "" || i.Hostname == "" || i.ConfigDir != "/tmp/x" || i.PID == 0 {
		t.Fatalf("%+v", i)
	}
	if !strings.Contains(i.Identity(), "@") || len(i.Rows()) != 10 {
		t.Fatal(i.Identity())
	}
	t.Logf("%+v", i)
}
