// Package host reports who and where the app is running, and offers file
// manager actions.
//
//   - Current(appID): the identity read by this process. Native builds read
//     the OS user (Source "native"). js/wasm builds have no host access
//     (Source "unavailable"), except with the remote build tag, where the
//     server's process identity is fetched from GET /api/host (Source
//     "server-process").
//   - Server(appID): the identity of this process labeled "server-process",
//     for server-side web targets that render it into a browser UI.
//
// On macOS, os/user may not return the full name in builds without cgo;
// Current then leaves FullName empty and explains it in Notes.
package host

import "errors"

// Source says where host information came from.
type Source string

const (
	SourceNative      Source = "native"         // read by this process on the user's own machine
	SourceServer      Source = "server-process" // read by a server process (web targets)
	SourceUnavailable Source = "unavailable"
)

// Info describes the running user and machine.
type Info struct {
	Source    Source   `json:"source"`
	Username  string   `json:"username"`
	FullName  string   `json:"fullName"`
	UID       string   `json:"uid"`
	HomeDir   string   `json:"homeDir"`
	Hostname  string   `json:"hostname"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	PID       int      `json:"pid"`
	ConfigDir string   `json:"configDir"`
	Locale    string   `json:"locale"`
	Notes     []string `json:"notes"`
}

// Identity returns "<Username>@<Hostname>", or "unknown user" when the
// username is unavailable.
func (i Info) Identity() string {
	if i.Username == "" {
		return "unknown user"
	}
	if i.Hostname == "" {
		return i.Username
	}
	return i.Username + "@" + i.Hostname
}

// Available reports whether host actions (reveal, open) can run on the
// user's machine: only for Source "native".
func (i Info) Available() bool { return i.Source == SourceNative }

// Row is one label/value line of the host panel.
type Row struct{ Label, Value string }

// Rows returns the host panel lines in display order, with "(unavailable)"
// for empty values. Notes are not included.
func (i Info) Rows() []Row {
	v := func(s string) string {
		if s == "" {
			return "(unavailable)"
		}
		return s
	}
	osArch := ""
	if i.OS != "" || i.Arch != "" {
		osArch = i.OS + " / " + i.Arch
	}
	pid := ""
	if i.PID > 0 {
		pid = itoa(i.PID)
	}
	return []Row{
		{"Source", v(string(i.Source))},
		{"Username", v(i.Username)},
		{"Full name", v(i.FullName)},
		{"UID", v(i.UID)},
		{"Home directory", v(i.HomeDir)},
		{"Hostname", v(i.Hostname)},
		{"OS / Arch", v(osArch)},
		{"PID", v(pid)},
		{"Config directory", v(i.ConfigDir)},
		{"Locale", v(i.Locale)},
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// ErrUnavailable is returned by host actions in builds without host access.
var ErrUnavailable = errors.New("not available in the browser")
