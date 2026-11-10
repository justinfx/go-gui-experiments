//go:build remote && !js

package backend

import (
	"errors"
	"os"
)

func baseURL() (string, error) {
	if u := os.Getenv("GGUIX_REMOTE_URL"); u != "" {
		return u, nil
	}
	return "", errors.New("GGUIX_REMOTE_URL is required in remote builds")
}
