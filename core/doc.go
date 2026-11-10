// Package core holds the shared domain model for the GGUIX application:
// item types, queries and paging, validation, display formatting, the
// client-side Pager, preferences types, custom-rendering colors and
// bench/metric helpers.
//
// Sub-packages:
//
//   - gguix/core/memory: in-process DataService over a generated dataset
//   - gguix/core/remote: HTTP client DataService/PrefsStore (talks to gguix/core/server)
//   - gguix/core/prefs: file and browser localStorage PrefsStore implementations
//   - gguix/core/backend: Open picks memory or remote by build tag
//   - gguix/core/host: host user information and file manager actions
//   - gguix/core/bench: the bench script runner for Go UIs
//   - gguix/core/server and gguix/core/cmd/server: the HTTP API server
//
// Everything in this module is pure Go and builds for GOOS=js GOARCH=wasm,
// except the file selected by the cgoprobe build tag.
package core
