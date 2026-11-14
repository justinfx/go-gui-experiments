# GGUIX go-gui native UI app. See README.md.
SHELL := /bin/bash
APP := gguix-gogui
BUNDLE := GGUIX-gogui.app
GOGUI := $(shell go list -m -f '{{.Dir}}' github.com/go-gui-org/go-gui)
BUILDAPP := go run github.com/go-gui-org/go-gui/cmd/buildapp@v0.96.0
SERVE_PORT := 8090
SERVER := cd ../../core && go run ./cmd/server -static ../apps/gogui/bin/web -addr :$(SERVE_PORT)

.PHONY: deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

# go-gui is a plain Go module; macOS needs the Xcode Command Line Tools (cgo, Metal).
deps:
	go mod download

# gguix:target desktop
# go-gui has no hot reload; the loop is go run.
dev:
	go run .

build:
	@mkdir -p bin
	go build -o bin/$(APP) .

package: build
	rm -rf bin/$(BUNDLE) bin/pkg
	$(BUILDAPP) -o bin/pkg -name GGUIX-gogui -id dev.gguix.gogui bin/$(APP)
	mv bin/pkg/*.app bin/$(BUNDLE)
	rm -rf bin/pkg

run: build
	./bin/$(APP)

bench: build
	@d=$$(mktemp -d); GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$d GGUIX_BENCH_EXPORT=$$d/bench-export.csv ./bin/$(APP) 2>&1 | grep -E 'GGUIX_METRIC|rror'; true

cross-windows:
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(APP)-windows-amd64.exe .

cross-linux:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/$(APP)-linux-amd64 .
# gguix:end

# gguix:target web
define webbuild
	rm -rf bin/web
	mkdir -p bin/web
	GOOS=js GOARCH=wasm go build $(1) -o bin/web/main.wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" bin/web/
	cp "$(GOGUI)/gui/backend/web/gogui.js" bin/web/
	cp web/index.html bin/web/
endef

web:
	$(call webbuild,)
	$(SERVER)

web-remote:
	$(call webbuild,-tags remote)
	$(SERVER)

bench-web:
	$(call webbuild,)
	@echo "open http://localhost:$(SERVE_PORT)/?bench=1"
	$(SERVER)
# gguix:end

test:
	@./run_tests.sh

cgoprobe:
	@mkdir -p bin; \
	if CGO_ENABLED=1 go build -tags cgoprobe -o bin/probe-desktop . >bin/probe-desktop.log 2>&1; then \
	  echo "CGOPROBE desktop PASS"; else echo "CGOPROBE desktop FAIL $$(tail -c 300 bin/probe-desktop.log | tr '\n' ' ')"; fi; \
	if GOOS=js GOARCH=wasm go build -tags cgoprobe -o bin/probe-web.wasm . >bin/probe-web.log 2>&1; then \
	  echo "CGOPROBE web PASS"; else echo "CGOPROBE web FAIL $$(tail -c 300 bin/probe-web.log | tr '\n' ' ')"; fi; \
	if GOOS=js GOARCH=wasm go build -tags remote -o bin/probe-wr.wasm . >bin/probe-wr.log 2>&1 && \
	   (cd ../../core && CGO_ENABLED=1 go build -tags cgoprobe -o ../apps/gogui/bin/probe-server ./cmd/server) >>bin/probe-wr.log 2>&1; then \
	  echo "CGOPROBE web-remote PASS wasm client without cgoprobe, server with cgoprobe"; \
	else echo "CGOPROBE web-remote FAIL $$(tail -c 300 bin/probe-wr.log | tr '\n' ' ')"; fi; true

clean:
	rm -rf bin
