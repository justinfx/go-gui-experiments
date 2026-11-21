# Toolchain: Gophics

| Name | Version | Install command | Approx. size | Needed for | cgo-related |
|---|---|---|---|---|---|
| Go | go1.27.3 darwin/arm64 | (preinstalled) | n/a | all | no |
| `gophics` CLI | `v0.0.0-20261103180631-8d360971eb89` | `GOBIN=apps/gophics/.tools go install github.com/doug/gophics/cmd/gophics@8d36097` (`make deps`) | 16 MB binary | dev, build (desktop and web), package | no |
| Go modules (gophics and deps: go-text/typesetting, go-webgpu/goffi, x/image, x/net, x/sys, x/text, x/tools, audio decoders) | per `go.sum` | `go mod download` | about 60 MB in the module cache | build | no (goffi does dlopen-style FFI in pure Go, no cgo) |
| `wasm_exec.js` | from the Go toolchain, copied by `gophics build -p web` | none | 17 KB | web | no |
| Google Chrome (WebGPU) | installed | none | n/a | web runtime | no |
| C compiler (Xcode CLT clang) | installed | none | n/a | `cgoprobe` only | yes |
| Docker + `gguix-linux` image | existing | `linux/dx.sh build` | 3.9 GB image | Linux runtime check only | no |

No global installs were made. Cross-compiling to Linux and Windows needs nothing beyond Go.

## Clean machine steps

Desktop:

```sh
cd apps/gophics
make deps
make run
```

Web (in-process core), then web-remote:

```sh
make web          # open http://localhost:8090 in Chrome
make web-remote   # same URL, data from core/cmd/server
```
