//go:build !js

package host

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"gguix/core/prefs"
)

// Current returns this process's host identity with Source "native".
func Current(appID string) Info {
	i := read(appID)
	i.Source = SourceNative
	return i
}

// Server returns this process's host identity with Source "server-process".
func Server(appID string) Info {
	i := read(appID)
	i.Source = SourceServer
	return i
}

func read(appID string) Info {
	i := Info{OS: runtime.GOOS, Arch: runtime.GOARCH, PID: os.Getpid(), ConfigDir: prefs.Dir(appID)}
	if u, err := user.Current(); err == nil {
		i.Username, i.FullName, i.UID, i.HomeDir = u.Username, u.Name, u.Uid, u.HomeDir
	} else {
		i.Notes = append(i.Notes, "os/user failed: "+err.Error())
		i.Username = os.Getenv("USER")
		i.HomeDir, _ = os.UserHomeDir()
	}
	if i.FullName == "" || i.FullName == i.Username {
		i.FullName = ""
		i.Notes = append(i.Notes, "full name unavailable in this build")
	}
	if h, err := os.Hostname(); err == nil {
		i.Hostname = h
	} else {
		i.Notes = append(i.Notes, "hostname unavailable: "+err.Error())
	}
	// Screenshot-only override, so captures don't show the real machine:
	// GGUIX_SHOT_IDENTITY=user@host replaces the displayed username and hostname.
	if v := os.Getenv("GGUIX_SHOT_IDENTITY"); v != "" {
		if u, h, ok := strings.Cut(v, "@"); ok {
			i.Username, i.Hostname, i.FullName = u, h, ""
			i.Notes = append(i.Notes, "identity overridden by GGUIX_SHOT_IDENTITY")
		}
	}
	i.Locale = os.Getenv("LC_ALL")
	if i.Locale == "" {
		i.Locale = os.Getenv("LANG")
	}
	return i
}

// RevealInFileManager shows path selected in the OS file manager
// (macOS: open -R; Linux: xdg-open on the parent directory;
// Windows: explorer /select,).
func RevealInFileManager(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", abs).Run()
	case "windows":
		return exec.Command("explorer", "/select,"+abs).Run()
	default:
		dir := abs
		if !st.IsDir() {
			dir = filepath.Dir(abs)
		}
		return exec.Command("xdg-open", dir).Run()
	}
}

// OpenWithDefaultApp opens path with the OS default application.
func OpenWithDefaultApp(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Run()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path).Run()
	case "linux", "freebsd":
		return exec.Command("xdg-open", path).Run()
	}
	return fmt.Errorf("open not supported on %s", runtime.GOOS)
}
