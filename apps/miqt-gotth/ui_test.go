package gotth

import (
	"strings"
	"testing"
)

func TestSplitAndColumnResizeMarkup(t *testing.T) {
	e := newEnv(t, 0, nil)
	body := e.do("GET", "/", nil).Body.String()
	for _, want := range []string{`data-slot="resizable-panel-group"`, `data-slot="resizable-handle"`, `id="divider"`, `data-col-resize="0"`, `data-col-resize="10"`} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %s", want)
		}
	}
}

func TestNativeMenuHidesInPageMenubar(t *testing.T) {
	e := newEnv(t, 0, &fakeNative{})
	if body := e.do("GET", "/", nil).Body.String(); !strings.Contains(body, `id="app-menubar"`) {
		t.Fatal("in-page menu bar missing in the default desktop build")
	}
	e.srv.opts.NativeMenu = true
	body := e.do("GET", "/", nil).Body.String()
	if strings.Contains(body, `id="app-menubar"`) || !strings.Contains(body, `data-native-menu="true"`) {
		t.Fatal("native menu build still draws the in-page menu bar")
	}
}
