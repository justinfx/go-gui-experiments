# GGUIX IUP-Go app. See README.md.
SHELL := /bin/bash
comma := ,
APP := gguix-iupgo
BUNDLE := GGUIX-iupgo.app
IUPKG := go run github.com/gen2brain/iup-go/cmd/iupkg@cbcdf09
SERVE_PORT := 8090
# TABLE selects the desktop table: matrix (IupMatrixEx, drawn cells, default, scrolls much better)
# or native (IupTable with cell images). Both are tested the same way: make test TABLE=native
TABLE ?= matrix
ifeq ($(TABLE),matrix)
GOTAGS := ctrl,matrix
else
GOTAGS :=
endif
TAGFLAG := $(if $(GOTAGS),-tags $(GOTAGS),)

.PHONY: emsdk-install deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

# IUP-Go is a Go module with bundled C sources; macOS needs the Xcode Command Line Tools (cgo, Cocoa).
deps:
	go mod download

# gguix:target desktop
# IUP-Go has no hot reload; the loop is go run.
dev:
	go run $(TAGFLAG) .

build:
	@mkdir -p bin
	go build $(TAGFLAG) -o bin/$(APP) .

package:
	rm -rf bin/pkg bin/$(BUNDLE)
	$(IUPKG) package --os darwin --arch arm64 --name GGUIX-iupgo --id dev.gguix.iupgo --out bin/pkg $(if $(GOTAGS),--tags $(GOTAGS),) .
	mv bin/pkg/$(BUNDLE) bin/$(BUNDLE)

run: build
	./bin/$(APP)

bench: build
	@d=$$(mktemp -d); GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$d GGUIX_BENCH_EXPORT=$$d/bench-export.csv ./bin/$(APP) 2>&1 | grep -E 'GGUIX_METRIC|rror'; true

# CGO_ENABLED=0 loads IUP's bundled prebuilt library at run time.
cross-windows:
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(APP)-windows-amd64.exe .

cross-linux:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(APP)-linux-amd64 .
# gguix:end

# gguix:target web
# The web build needs the Emscripten-built IUP module (iup/external/wasm in the iup-go
# module). emsdk is installed outside the repo; the module is built from a writable copy
# because the Go module cache is read-only and the build script writes next to itself.
EMSDK ?= $(HOME)/emsdk
IUPWASM ?= $(HOME)/iupwasm
IUPDIR := $(shell go list -m -f '{{.Dir}}' github.com/gen2brain/iup-go/iup)
WASMDIR := $(IUPWASM)/repo/iup/external/wasm
SERVER := cd ../../core && go run ./cmd/server -static ../apps/iupgo/bin/web -coi -addr :$(SERVE_PORT)

# emsdk-install: one time, about 2 GB under $(EMSDK), see results/iupgo/toolchain.md.
emsdk-install:
	[ -d $(EMSDK) ] || git clone --depth 1 https://github.com/emscripten-core/emsdk.git $(EMSDK)
	cd $(EMSDK) && ./emsdk install latest && ./emsdk activate latest

$(WASMDIR)/build/iup.wasm:
	mkdir -p $(IUPWASM)/repo/iup && cp -R $(IUPDIR)/. $(IUPWASM)/repo/iup && chmod -R u+w $(IUPWASM)/repo
	. $(EMSDK)/emsdk_env.sh >/dev/null 2>&1 && cd $(WASMDIR) && bash ./build-wasm.sh -O -m

# The app runs in a Worker (IUP's loader); coi.js and coi-sw.js add the cross-origin isolation
# headers the core server does not send; gguix-main.js serves download, file pick and clipboard.
define webbuild
	mkdir -p bin/web
	GOOS=js GOARCH=wasm CGO_ENABLED=0 go build $(1) -o bin/web/app.wasm .
	cp $(WASMDIR)/build/iup.js $(WASMDIR)/build/iup.wasm $(WASMDIR)/web/worker.js $(WASMDIR)/web/iupwasm_dom.js bin/web/
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" bin/web/
	cp web/coi.js web/coi-sw.js web/gguix-main.js bin/web/
	echo '[]' > bin/web/resources.json
	sed -e 's#<head>#<head><script src="coi.js"></script><script src="gguix-main.js"></script>#' \
	    -e "s#var worker = new Worker('worker.js');#var worker = new Worker('worker.js' + location.search); gguixAttach(worker);#" \
	    $(WASMDIR)/web/index.html > bin/web/index.html
endef

web: $(WASMDIR)/build/iup.wasm
	$(call webbuild,)
	$(SERVER)

web-remote: $(WASMDIR)/build/iup.wasm
	$(call webbuild,-tags remote)
	$(SERVER)

bench-web: $(WASMDIR)/build/iup.wasm
	$(call webbuild,)
	@echo "open http://localhost:$(SERVE_PORT)/?bench=1"
	$(SERVER)
# gguix:end

test:
	@GOTAGS="$(GOTAGS)" ./run_tests.sh

cgoprobe:
	@mkdir -p bin; \
	if CGO_ENABLED=1 go build -tags "$(GOTAGS)$(if $(GOTAGS),$(comma),)cgoprobe" -o bin/probe-desktop . >bin/probe-desktop.log 2>&1; then \
	  echo "CGOPROBE desktop PASS"; else echo "CGOPROBE desktop FAIL $$(tail -c 300 bin/probe-desktop.log | tr '\n' ' ')"; fi; \
	if GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -tags cgoprobe -o bin/probe-web.wasm . >bin/probe-web.log 2>&1; then \
	  echo "CGOPROBE web PASS"; else echo "CGOPROBE web FAIL $$(tail -c 300 bin/probe-web.log | tr '\n' ' ')"; fi; \
	if GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -tags remote -o bin/probe-wr.wasm . >bin/probe-wr.log 2>&1 && \
	   (cd ../../core && CGO_ENABLED=1 go build -tags cgoprobe -o ../apps/iupgo/bin/probe-server ./cmd/server) >>bin/probe-wr.log 2>&1; then \
	  echo "CGOPROBE web-remote PASS wasm client without cgoprobe, server with cgoprobe"; \
	else echo "CGOPROBE web-remote FAIL $$(tail -c 300 bin/probe-wr.log | tr '\n' ' ')"; fi; true

clean:
	rm -rf bin
