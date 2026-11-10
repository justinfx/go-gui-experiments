package server_test

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gguix/core"
	"gguix/core/host"
	"gguix/core/memory"
	"gguix/core/remote"
	"gguix/core/server"
)

func setup(t *testing.T) (*memory.Service, *remote.Client) {
	t.Setenv("GGUIX_PREFS_DIR", t.TempDir())
	svc := memory.New(memory.Options{Rows: 2000, ExportRowsPerSec: -1})
	ts := httptest.NewServer(server.Handler(svc, ""))
	t.Cleanup(ts.Close)
	return svc, remote.New(ts.URL, "testapp")
}

func TestRemoteMatchesMemory(t *testing.T) {
	svc, c := setup(t)
	ctx := context.Background()
	q := core.Query{Search: "steel", Categories: []core.Category{core.Toys, core.Garden}, SortBy: core.SortPrice, Desc: true}
	want, _ := svc.Page(ctx, core.PageRequest{Query: q, Offset: 3, Limit: 50})
	got, err := c.Page(ctx, core.PageRequest{Query: q, Offset: 3, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != want.Total || len(got.Items) != len(want.Items) || got.Items[0].ID != want.Items[0].ID || got.QueryKey != want.QueryKey {
		t.Fatalf("mismatch: %d/%d %d/%d", got.Total, want.Total, len(got.Items), len(want.Items))
	}
	if !got.Items[0].UpdatedAt.Equal(want.Items[0].UpdatedAt) || got.Items[0].Name != want.Items[0].Name {
		t.Fatal("item fields differ")
	}
}

func TestRemoteGetUpdateValidation(t *testing.T) {
	_, c := setup(t)
	ctx := context.Background()
	it, err := c.Get(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	it.Name = "ab"
	_, err = c.Update(ctx, it)
	if v := core.AsValidationError(err); v == nil || v.For("name") != core.MsgName {
		t.Fatalf("err %v", err)
	}
	it.Name = "Renamed Thing"
	out, err := c.Update(ctx, it)
	if err != nil || out.Name != "Renamed Thing" {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, 99999); !errors.Is(err, core.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestRemoteExportAndPrefsAndHost(t *testing.T) {
	_, c := setup(t)
	ctx := context.Background()
	var buf bytes.Buffer
	var last [2]int
	n, err := c.Export(ctx, core.Query{Search: "oak"}, &buf, func(d, tot int) { last = [2]int{d, tot} })
	if err != nil || n == 0 || last != [2]int{n, n} {
		t.Fatalf("export n=%d err=%v last=%v", n, err, last)
	}
	if !strings.Contains(buf.String(), "# checksum ") {
		t.Fatal("no checksum line")
	}
	p := core.DefaultPrefs()
	p.Theme = core.ThemeDark
	if err := c.Save(p); err != nil {
		t.Fatal(err)
	}
	back, err := c.Load()
	if err != nil || back.Theme != core.ThemeDark {
		t.Fatalf("%v %+v", err, back)
	}
	hi, err := c.HostInfo(ctx)
	if err != nil || hi.Source != host.SourceServer || hi.Username == "" {
		t.Fatalf("%v %+v", err, hi)
	}
}

func TestRemoteExportCancel(t *testing.T) {
	t.Setenv("GGUIX_PREFS_DIR", t.TempDir())
	svc := memory.New(memory.Options{Rows: 2000, ExportRowsPerSec: 500})
	ts := httptest.NewServer(server.Handler(svc, ""))
	defer ts.Close()
	c := remote.New(ts.URL, "x")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(150*time.Millisecond, cancel)
	_, err := c.Export(ctx, core.Query{}, &bytes.Buffer{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v", err)
	}
}
