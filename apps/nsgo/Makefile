# GGUIX NSGo native UI app. See README.md.
SHELL := /bin/bash
APP := gguix-nsgo
BUNDLE := GGUIX-nsgo.app
NSGO := go tool nsgo
# nsgo build/dev put the app identity (nsgo.yaml name and id) into core.App
# with these linker flags; a plain go build needs them too, or the Linux
# window has no app id and notifications no app name.
CORE := github.com/nsgo-dev/nsgo/core
IDENT := -X '$(CORE).appName=GGUIX-nsgo' -X '$(CORE).appID=dev.gguix.nsgo' -X '$(CORE).appVersion=0.1.0' -X '$(CORE).appBuild=0.1.0'
SERVE_PORT := 8090
SERVER := cd ../../core && go run ./cmd/server -static ../apps/nsgo/bin/web -addr :$(SERVE_PORT)

.PHONY: deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

# NSGo is a plain Go module and its CLI is a go.mod tool directive: nothing
# beyond Go itself is installed.
deps:
	go mod download

# gguix:target desktop
# nsgo dev rebuilds and restarts the app on every saved Go file (hot reload).
dev:
	$(NSGO) dev

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "$(IDENT)" -o bin/$(APP) .

# nsgo build writes the .app (signed ad hoc); the tool names it after nsgo.yaml's name.
package:
	rm -rf bin/dist bin/$(BUNDLE)
	$(NSGO) build -target macos_arm64 -output bin/dist
	cp -R bin/dist/*-macos_arm64/GGUIX-nsgo.app bin/$(BUNDLE)

run: build
	./bin/$(APP)

bench: build
	@d=$$(mktemp -d); GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$d GGUIX_BENCH_EXPORT=$$d/bench-export.csv ./bin/$(APP) 2>&1 | grep -E 'GGUIX_METRIC|rror'; true

cross-windows:
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(IDENT)" -o bin/$(APP)-windows-amd64.exe .

cross-linux:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(IDENT)" -o bin/$(APP)-linux-amd64 .
# gguix:end

# gguix:target web
# nsgo build -target web writes the site (page, loader, wasm_exec.js, the
# wasm); GOFLAGS carries the build tag, because the tool has no tags flag.
define webbuild
	rm -rf bin/web bin/webdist
	GOFLAGS="$(1)" $(NSGO) build -target web -output bin/webdist
	mkdir -p bin/web
	cp -R bin/webdist/*-web/. bin/web/
endef

web:
	$(call webbuild,)
	$(SERVER)

web-remote:
	$(call webbuild,-tags=remote)
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
	   (cd ../../core && CGO_ENABLED=1 go build -tags cgoprobe -o ../apps/nsgo/bin/probe-server ./cmd/server) >>bin/probe-wr.log 2>&1; then \
	  echo "CGOPROBE web-remote PASS wasm client without cgoprobe, server with cgoprobe"; \
	else echo "CGOPROBE web-remote FAIL $$(tail -c 300 bin/probe-wr.log | tr '\n' ' ')"; fi; true

clean:
	rm -rf bin
