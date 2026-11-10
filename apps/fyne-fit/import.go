package main

import (
	"errors"
	"strings"

	"gguix/core"
)

// errTooLarge is shown when an imported notes file is over 64 KB.
var errTooLarge = errors.New("File is too large.")

// importNotesData puts at most the first 500 characters of data (UTF-8 text)
// into the Notes field. Files over 64 KB are rejected with an error dialog.
func (u *ui) importNotesData(data []byte) {
	if len(data) > core.MaxImportFileSize {
		u.showError(errTooLarge)
		return
	}
	if !u.form.has {
		return
	}
	text := strings.ToValidUTF8(string(data), "")
	r := []rune(text)
	if len(r) > core.MaxNotesRunes {
		r = r[:core.MaxNotesRunes]
	}
	u.form.SetNotes(string(r))
}
