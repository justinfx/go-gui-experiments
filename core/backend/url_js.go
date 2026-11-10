//go:build remote && js

package backend

import "syscall/js"

func baseURL() (string, error) {
	return js.Global().Get("location").Get("origin").String(), nil
}
