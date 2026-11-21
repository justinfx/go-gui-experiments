module gguix/apps/gophics

go 1.27.3

require (
	gguix/core v0.0.0-00010101000000-000000000000
	github.com/doug/gophics v0.0.0-20261103180631-8d360971eb89
	golang.org/x/image v0.44.0
)

require (
	github.com/go-text/typesetting v0.3.4 // indirect
	github.com/go-webgpu/goffi v0.6.1 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace gguix/core => ../../core
