# GGUIX MyGo native UI app. Run `make deps` first. See README.md.
SHELL := /bin/bash
APP := gguix-mygoui
MYGO := go tool mygo
OUT := build
BUNDLE := GGUIX-mygoui.app
SERVE_PORT := 8090
SERVER := cd ../../core && go run ./cmd/server -static ../apps/mygoui/bin/web -addr :$(SERVE_PORT)

.PHONY: deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

# The mygo CLI is pinned by the `tool` line in go.mod; nothing else to install.
deps:
	go mod download
	go build -o /dev/null github.com/egoist/mygo/cmd/mygo

dev:
	$(MYGO) dev

# gguix:target desktop
# mygo build writes build/darwin-<arch>/GGUIX-mygoui.app; the bare
# executable inside it is copied to bin/.
build:
	@mkdir -p bin
	$(MYGO) build -skip-dmg
	@exe=$$(ls -d $(OUT)/darwin-*/$(BUNDLE)/Contents/MacOS/* $(OUT)/linux-*/gguix-mygoui 2>/dev/null | head -1); \
	test -n "$$exe" || { echo "mygo build produced no executable"; exit 1; }; \
	rm -f bin/$(APP); cp "$$exe" bin/$(APP); echo "built bin/$(APP)"

package:
	@mkdir -p bin
	$(MYGO) build -skip-dmg
	rm -rf bin/$(BUNDLE)
	cp -R $(OUT)/darwin-*/$(BUNDLE) bin/$(BUNDLE)

run: build
	./bin/$(APP)

bench: build
	@d=$$(mktemp -d); GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$d GGUIX_BENCH_EXPORT=$$d/bench-export.csv ./bin/$(APP) 2>&1 | grep -E 'GGUIX_METRIC|rror'; true

cross-windows:
	$(MYGO) build -skip-dmg -platform windows/amd64

cross-linux:
	$(MYGO) build -skip-dmg -platform linux/amd64
# gguix:end

# gguix:target web
# mygo build -platform web writes <out>/web (index.html, wasm_exec.js, the
# wasm). The core server serves it and the remote API.
define webbuild
	rm -rf bin/webout bin/web
	$(1) $(MYGO) build -platform web -o bin/webout
	mv bin/webout/web bin/web && rmdir bin/webout
endef

web:
	$(call webbuild,)
	$(SERVER)

web-remote:
	$(call webbuild,GOFLAGS=-tags=remote)
	$(SERVER)

bench-web:
	$(call webbuild,)
	@echo "open http://localhost:$(SERVE_PORT)/?bench=1"
	$(SERVER)
# gguix:end

test:
	@./run_tests.sh

# CGO_ENABLED=1 lets cgo code build (mygo defaults to CGO_ENABLED=0); the
# tag travels in GOFLAGS through mygo build.
cgoprobe:
	@mkdir -p bin; \
	if CGO_ENABLED=1 GOFLAGS=-tags=cgoprobe $(MYGO) build -skip-dmg -o bin/probe-out >bin/probe-desktop.log 2>&1; then \
	  echo "CGOPROBE desktop PASS builds with CGO_ENABLED=1 and -tags cgoprobe"; \
	else echo "CGOPROBE desktop FAIL $$(tail -c 300 bin/probe-desktop.log | tr '\n' ' ')"; fi; \
	if GOFLAGS=-tags=cgoprobe $(MYGO) build -platform web -o bin/probe-web >bin/probe-web.log 2>&1; then \
	  echo "CGOPROBE web PASS"; \
	else echo "CGOPROBE web FAIL $$(tail -c 300 bin/probe-web.log | tr '\n' ' ')"; fi; \
	if $(MYGO) build -platform web -o bin/probe-webremote >bin/probe-wr-client.log 2>&1 && \
	   (cd ../../core && CGO_ENABLED=1 go build -tags cgoprobe -o ../apps/mygoui/bin/probe-server ./cmd/server) >bin/probe-wr-server.log 2>&1; then \
	  echo "CGOPROBE web-remote PASS wasm client built without cgoprobe, server built with cgoprobe"; \
	else echo "CGOPROBE web-remote FAIL $$(tail -c 300 bin/probe-wr-*.log | tr '\n' ' ')"; fi; true

clean:
	rm -rf bin build .mygo
