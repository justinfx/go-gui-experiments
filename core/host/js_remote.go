//go:build js && remote

package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"syscall/js"
	"time"
)

// Current fetches the server's process identity from GET /api/host on the
// page origin (Source "server-process"). Do not call it from inside a
// syscall/js callback (it blocks on the network); call it from a goroutine.
// On failure it returns Source "unavailable" with the error in Notes.
func Current(appID string) Info {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	origin := js.Global().Get("location").Get("origin").String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/api/host?app="+url.QueryEscape(appID), nil)
	if err != nil {
		return failed(appID, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return failed(appID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return failed(appID, fmt.Errorf("GET /api/host: %s", resp.Status))
	}
	var i Info
	if err := json.NewDecoder(resp.Body).Decode(&i); err != nil {
		return failed(appID, err)
	}
	i.Source = SourceServer
	return i
}

// Server returns Source "unavailable" in js/wasm builds.
func Server(appID string) Info { return unavailable(appID) }

func failed(appID string, err error) Info {
	i := unavailable(appID)
	i.Notes = append(i.Notes, "host request failed: "+err.Error())
	return i
}
