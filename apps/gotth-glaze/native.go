package gotth

// Native is the set of operating system features that a plain browser tab
// gets from the browser itself but the desktop shell must provide from Go.
// cmd/desktop supplies an implementation; cmd/web passes nil.
type Native interface {
	// SaveDialog shows a native save dialog. ok is false when cancelled.
	SaveDialog(defaultName string) (path string, ok bool, err error)
	// OpenDialog shows a native open dialog for files with the extension.
	OpenDialog(ext string) (path string, ok bool, err error)
	// Notify sends an OS notification.
	Notify(title, body string) error
	// CopyText puts text on the system clipboard.
	CopyText(text string) error
}
