module gguix/apps/nsgo

go 1.27.3

require (
	gguix/core v0.0.0-00010101000000-000000000000
	github.com/ebitengine/purego v0.11.3
	github.com/nsgo-dev/nsgo v0.0.0-20261114190341-2345b2fa5aba
)

require (
	github.com/andybalholm/brotli v1.3.0 // indirect
	github.com/dop251/goja v0.0.0-20261027234916-5f913d1693d9 // indirect
	github.com/evanw/esbuild v0.28.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/yuin/goldmark v1.8.7 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/image v0.48.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	rsc.io/qr v0.2.0 // indirect
)

replace gguix/core => ../../core

tool github.com/nsgo-dev/nsgo/cmd/nsgo
