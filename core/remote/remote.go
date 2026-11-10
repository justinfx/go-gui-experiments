// Package remote is an HTTP client for the gguix/core/server API. A Client
// implements core.DataService and core.PrefsStore, so an app built with the
// remote tag can use it in place of the in-process service (see
// gguix/core/backend). It works in native and js/wasm builds; in js/wasm,
// do not call it from inside a syscall/js callback (calls block on the
// network), call it from a goroutine.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gguix/core"
	"gguix/core/host"
)

// Client talks to a gguix server at BaseURL.
type Client struct {
	base  string
	appID string
	http  *http.Client
}

var (
	_ core.DataService = (*Client)(nil)
	_ core.PrefsStore  = (*Client)(nil)
)

// New returns a client for baseURL (for example "http://localhost:8090")
// whose prefs are stored under appID.
func New(baseURL, appID string) *Client {
	return &Client{base: strings.TrimRight(baseURL, "/"), appID: appID, http: &http.Client{}}
}

// BaseURL returns the server URL.
func (c *Client) BaseURL() string { return c.base }

type apiError struct {
	Error string `json:"error"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return core.ErrNotFound
	case resp.StatusCode == http.StatusUnprocessableEntity:
		var ve core.ValidationError
		if err := json.NewDecoder(resp.Body).Decode(&ve); err != nil {
			return err
		}
		return &ve
	case resp.StatusCode >= 400:
		var ae apiError
		_ = json.NewDecoder(resp.Body).Decode(&ae)
		if ae.Error == "" {
			ae.Error = resp.Status
		}
		return fmt.Errorf("%s %s: %s", method, path, ae.Error)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Page calls GET /api/items.
func (c *Client) Page(ctx context.Context, req core.PageRequest) (core.Page, error) {
	v := req.Query.Values()
	v.Set("offset", strconv.Itoa(req.Offset))
	v.Set("limit", strconv.Itoa(req.Limit))
	var p core.Page
	err := c.do(ctx, http.MethodGet, "/api/items?"+v.Encode(), nil, &p)
	if p.Items == nil {
		p.Items = []core.Item{}
	}
	return p, err
}

// Get calls GET /api/items/{id}.
func (c *Client) Get(ctx context.Context, id int64) (core.Item, error) {
	var it core.Item
	err := c.do(ctx, http.MethodGet, "/api/items/"+strconv.FormatInt(id, 10), nil, &it)
	return it, err
}

// Update calls PUT /api/items/{id}.
func (c *Client) Update(ctx context.Context, it core.Item) (core.Item, error) {
	var out core.Item
	err := c.do(ctx, http.MethodPut, "/api/items/"+strconv.FormatInt(it.ID, 10), it, &out)
	return out, err
}

// Export calls GET /api/export and copies the CSV stream to w, reporting
// progress from the X-Total-Count header and the rows received.
func (c *Client) Export(ctx context.Context, q core.Query, w io.Writer, progress func(done, total int)) (int, error) {
	if progress == nil {
		progress = func(int, int) {}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/export?"+q.Values().Encode(), nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("GET /api/export: %s", resp.Status)
	}
	total, _ := strconv.Atoi(resp.Header.Get("X-Total-Count"))
	progress(0, total)
	buf := make([]byte, 32<<10)
	lines, rows := 0, 0
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return rows, werr
			}
			lines += bytes.Count(buf[:n], []byte{'\n'})
			rows = max(0, min(total, lines-1)) // minus the header line
			if time.Since(last) >= 100*time.Millisecond {
				last = time.Now()
				progress(rows, total)
			}
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			if ctx.Err() != nil {
				return rows, ctx.Err()
			}
			return rows, rerr
		}
	}
	progress(total, total)
	return total, nil
}

// Load calls GET /api/prefs/{appID}.
func (c *Client) Load() (core.Prefs, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p := core.DefaultPrefs()
	if err := c.do(ctx, http.MethodGet, "/api/prefs/"+url.PathEscape(c.appID), nil, &p); err != nil {
		return core.DefaultPrefs(), err
	}
	return p.Sanitize(), nil
}

// Save calls PUT /api/prefs/{appID}.
func (c *Client) Save(p core.Prefs) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.do(ctx, http.MethodPut, "/api/prefs/"+url.PathEscape(c.appID), p, nil)
}

// HostInfo calls GET /api/host and returns the server's process identity.
func (c *Client) HostInfo(ctx context.Context) (host.Info, error) {
	var i host.Info
	err := c.do(ctx, http.MethodGet, "/api/host?app="+url.QueryEscape(c.appID), nil, &i)
	return i, err
}
