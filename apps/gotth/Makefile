# GGUIX GoTTH. Run `make deps` once, then any other target.
SHELL := /bin/bash
GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(CURDIR)/bin/tools:$(PATH)

TEMPL_VERSION    := v0.3.1100
SHADCN_VERSION   := v2.0.0
TAILWIND_VERSION := v4.3.5
HTMX_VERSION     := 2.0.11
HTMX_SSE_VERSION := 2.2.4

TAILWIND_ASSET := $(shell if [ "$$(uname -m)" = arm64 ]; then echo macos-arm64; else echo macos-x64; fi)
BIN := bin
APP := $(BIN)/GGUIX-gotth.app
CSS_IN  := assets/css/globals.css
CSS_OUT := assets/css/output.css

.PHONY: deps gen dev build build-native package-native run-native bench-native package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

deps:
	@mkdir -p $(BIN)/tools static/htmx
	@command -v templ >/dev/null && templ version | grep -q $(TEMPL_VERSION) || go install github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION)
	@command -v shadcn-templ >/dev/null || go install github.com/axadrn/shadcn-templ/v2/cmd/shadcn-templ@$(SHADCN_VERSION)
	@if ! $(BIN)/tools/tailwindcss --help 2>/dev/null | grep -q $(TAILWIND_VERSION); then \
	  curl -sSfL -o $(BIN)/tools/tailwindcss https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_ASSET) && chmod +x $(BIN)/tools/tailwindcss; fi
	@test -s static/htmx/htmx.min.js || curl -sSfL -o static/htmx/htmx.min.js https://cdn.jsdelivr.net/npm/htmx.org@$(HTMX_VERSION)/dist/htmx.min.js
	@test -s static/htmx/sse.js || curl -sSfL -o static/htmx/sse.js https://cdn.jsdelivr.net/npm/htmx-ext-sse@$(HTMX_SSE_VERSION)/dist/sse.js
	@go mod download
	@echo "deps ok"

# Generated files: templ components and the Tailwind stylesheet.
gen:
	@templ generate 2>&1 | grep -v '^(✓)' || true
	@$(BIN)/tools/tailwindcss -i $(CSS_IN) -o $(CSS_OUT) --minify 2>&1 | tail -1

dev:
	@echo "dev: templ watches the .templ files and proxies http://localhost:7331 to the app on :8090; static files are read from disk."
	@$(BIN)/tools/tailwindcss -i $(CSS_IN) -o $(CSS_OUT) --watch > /dev/null 2>&1 & \
	  trap 'kill %1 2>/dev/null' EXIT; \
	  GOTTH_DEV=1 templ generate --watch --proxy=http://localhost:8090 --cmd="go run ./cmd/web" --open-browser=false

# gguix:target desktop
build: gen
	@mkdir -p $(BIN)
	go build -trimpath -ldflags "-s -w" -o $(BIN)/gguix-gotth ./cmd/desktop

# The desktop framework (webview_go) has no packaging tool, so the bundle is
# assembled by hand: binary, Info.plist and nothing else.
package: build
	@mkdir -p $(APP)/Contents/MacOS
	@cp $(BIN)/gguix-gotth $(APP)/Contents/MacOS/GGUIX-gotth
	@cp desktop/Info.plist $(APP)/Contents/Info.plist
	@echo "packaged $(APP)"

run: build
	$(BIN)/gguix-gotth

bench: build
	@d=$$(mktemp -d /tmp/gotth-bench.XXXXXX); \
	GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$d GGUIX_BENCH_EXPORT=$$d/export.csv $(BIN)/gguix-gotth

# Native shim build (macOS only, -tags native): NSMenu menu bar, NSOpenPanel and
# NSSavePanel, user notification and NSPasteboard through cgo/Objective-C
# (cmd/desktop/native_shim.*). The default build keeps the in-page menu bar and
# the osascript/pbcopy helpers.
build-native: gen
	@mkdir -p $(BIN)
	go build -tags native -trimpath -ldflags "-s -w" -o $(BIN)/gguix-gotth-native ./cmd/desktop

# UNUserNotificationCenter needs a bundle, so notifications from the packaged
# app use it; the bare binary falls back to NSUserNotificationCenter.
package-native: build-native
	@mkdir -p $(BIN)/GGUIX-gotth-native.app/Contents/MacOS
	@cp $(BIN)/gguix-gotth-native $(BIN)/GGUIX-gotth-native.app/Contents/MacOS/GGUIX-gotth
	@cp desktop/Info.plist $(BIN)/GGUIX-gotth-native.app/Contents/Info.plist
	@echo "packaged $(BIN)/GGUIX-gotth-native.app"

run-native: build-native
	$(BIN)/gguix-gotth-native

bench-native: build-native
	@d=$$(mktemp -d /tmp/gotth-bench.XXXXXX); \
	GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$d GGUIX_BENCH_EXPORT=$$d/export.csv $(BIN)/gguix-gotth-native
# gguix:end

# gguix:target web
web: gen
	@mkdir -p $(BIN)
	go build -trimpath -ldflags "-s -w" -o $(BIN)/gguix-gotth-web ./cmd/web
	$(BIN)/gguix-gotth-web

web-remote:
	@echo "NOTE: the core already runs on the server in this stack, so web-remote is the same as web."
	@$(MAKE) --no-print-directory web

bench-web:
	@echo "Open http://localhost:8090/?bench=1 in a browser; the metrics print to the browser console."
	@$(MAKE) --no-print-directory web
# gguix:end

# Runs every test and ends with the GGUIX_TESTS line. T9 needs Node and Chrome.
test: gen
	@set -o pipefail; go test -count=1 -v ./... 2>&1 | tee $(BIN)/test.log | grep -E '^(--- (FAIL|SKIP)|FAIL|panic)' ; \
	st=$${PIPESTATUS[0]}; \
	line=$$(grep '^GGUIX_TESTS' $(BIN)/test.log | tail -1); \
	echo "$${line:-GGUIX_TESTS passed=0 failed=1 not_feasible=0}"; \
	exit $$st

# Rebuild each buildable target with -tags cgoprobe. Never fails the make run.
cgoprobe: gen
	@r=$$(go build -tags cgoprobe -o /dev/null ./cmd/web 2>&1) && echo "CGOPROBE web PASS" || echo "CGOPROBE web FAIL $$(echo "$$r" | head -1)"
	@r=$$(go build -tags cgoprobe -o /dev/null ./cmd/desktop 2>&1 | grep -E 'error|undefined' | head -1); \
	  if [ -z "$$r" ]; then echo "CGOPROBE desktop PASS"; else echo "CGOPROBE desktop FAIL $$r"; fi
	@r=$$(go test -tags cgoprobe -count=1 -run 'T8' . 2>&1 | tail -1); case "$$r" in ok*) echo "CGOPROBE web-export-test PASS";; *) echo "CGOPROBE web-export-test FAIL $$r";; esac
	@true

cross-windows: gen
	@./desktop/cross.sh windows

cross-linux: gen
	@./desktop/cross.sh linux

clean:
	@find $(BIN) -mindepth 1 -not -path '$(BIN)/tools*' -delete 2>/dev/null || true
	@find . -name '*_templ.go' -delete
	@find assets/css -name output.css -delete
	@echo clean
