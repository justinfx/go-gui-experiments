# GGUIX Wails app. Run `make deps` first. See README.md.
SHELL := /bin/bash
export PATH := $(shell go env GOPATH)/bin:$(PATH)
WAILS_VERSION := v3.1.0
APP := gguix-wails
WEB_PORT := 8090

.PHONY: deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

deps:
	go install github.com/wailsapp/wails/v3/cmd/wails3@$(WAILS_VERSION)
	cd frontend && npm ci
	go mod download

dev:
	wails3 dev -config ./build/config.yml -port 9245

# gguix:target desktop
build:
	wails3 task build

package:
	wails3 task package
	mv bin/$(APP).app bin/rename-tmp.app
	mv bin/rename-tmp.app bin/GGUIX-wails.app
	codesign --force --deep --sign - bin/GGUIX-wails.app

run:
	wails3 task run

bench: build
	@d=$$(mktemp -d); GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$d GGUIX_BENCH_EXPORT=$$d/bench-export.csv wails3 task run 2>&1 | grep -E 'GGUIX_METRIC|rror'; true
# gguix:end

# gguix:target web
web:
	wails3 task build:server
	WAILS_SERVER_PORT=$(WEB_PORT) ./bin/$(APP)-server

web-remote:
	@echo "NOTE: the core already runs on the server in Wails server mode, so web-remote is the same as web."
	@$(MAKE) --no-print-directory web

bench-web:
	wails3 task build:server
	@echo "Open http://localhost:$(WEB_PORT)/?bench=1 in a browser; metrics print to the browser console."
	WAILS_SERVER_PORT=$(WEB_PORT) GGUIX_PREFS_DIR=$$(mktemp -d) ./bin/$(APP)-server
# gguix:end

test:
	@mkdir -p bin
	@# Stale result files must not be scored (audit C5).
	rm -f bin/test-go.log bin/test-vitest.json frontend/e2e/results.json
	wails3 task build:server >/dev/null
	go test -v -count=1 ./... 2>&1 | grep -v '^ld:' | tee bin/test-go.log | grep -E '^(--- |FAIL|ok|panic)' || true
	cd frontend && npx vitest run --reporter=json --outputFile=../bin/test-vitest.json >/dev/null 2>&1 || true
	cd frontend && npx playwright test || true
	@for f in bin/test-go.log bin/test-vitest.json frontend/e2e/results.json; do test -s $$f || { echo "missing test result file $$f"; echo "GGUIX_TESTS passed=0 failed=1 not_feasible=0"; exit 1; }; done
	@python3 scripts/gguix_tests.py 0

# Rebuild every buildable target with -tags cgoprobe (BRIEF 4.10).
cgoprobe:
	@mkdir -p bin; (cd frontend && npm run build -s >/dev/null 2>&1); \
	if go build -tags production,cgoprobe -o bin/probe-desktop . 2>bin/probe-desktop.err; then \
	  if GGUIX_PREFS_DIR=$$(mktemp -d) go test -tags cgoprobe -count=1 -run 'TestExportStreamWriter|TestExportCancelRemovesFile' . >/dev/null 2>&1; \
	  then echo "CGOPROBE desktop PASS builds and the export code runs with the C checksum"; \
	  else echo "CGOPROBE desktop FAIL builds but the export tests fail"; fi; \
	else echo "CGOPROBE desktop FAIL $$(head -c 200 bin/probe-desktop.err | tr '\n' ' ')"; fi; \
	if go build -tags server,production,cgoprobe -o bin/probe-web . 2>bin/probe-web.err; then \
	  ( GGUIX_ROWS=50 GGUIX_LATENCY_MS=0 GGUIX_PREFS_DIR=$$(mktemp -d) WAILS_SERVER_PORT=8192 bin/probe-web >/dev/null 2>&1 & echo $$! >bin/probe.pid ); sleep 2; \
	  if curl -sf localhost:8192/health >/dev/null; then echo "CGOPROBE web PASS builds and serves"; else echo "CGOPROBE web FAIL built but did not start"; fi; \
	  kill $$(cat bin/probe.pid) 2>/dev/null; \
	else echo "CGOPROBE web FAIL $$(head -c 200 bin/probe-web.err | tr '\n' ' ')"; fi; true

cross-windows:
	wails3 task setup:docker
	wails3 task build GOOS=windows ARCH=amd64

# Apple Silicon Docker runs arm64 images, but the template's linux/amd64 build uses plain gcc, so
# the image is built for linux/amd64 (emulated) and the template's entrypoint is run directly.
cross-linux:
	docker build --platform linux/amd64 -t wails-cross-amd64 -f build/docker/Dockerfile.cross build/docker
	cd frontend && npm run build -s
	mkdir -p bin
	docker run --rm --platform linux/amd64 -v "$$PWD:/app" -v "$$HOME/go/pkg/mod:/go/pkg/mod" -v "$$PWD/../../core:/core:ro" -e APP_NAME=$(APP) wails-cross-amd64 linux amd64

clean:
	rm -rf bin
	rm -rf frontend/dist frontend/bindings
	rm -f frontend/e2e/results.json frontend/e2e/oracle.json
