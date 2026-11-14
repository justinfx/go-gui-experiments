# GGUIX MyGo app (web frontend mode). Run `make deps` first. See README.md.
SHELL := /bin/bash
APP := gguix-mygo
MYGO := frontend/node_modules/.bin/mygo
OUT := build/out

.PHONY: deps deps-check dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

deps:
	cd frontend && npm ci
	go mod download

deps-check:
	@test -x $(MYGO) || { echo "run make deps first"; exit 1; }

dev: deps-check
	$(MYGO) dev

# gguix:target desktop
# mygo build runs the frontend build, regenerates the TypeScript client,
# embeds frontend/dist and writes an app bundle (macOS) or packages (Linux).
# On macOS a web-frontend MyGo app opens no window when its executable runs
# outside the .app bundle, so bin/ holds a symlink into the bundle, not a copy.
build: deps-check
	@mkdir -p bin
	$(MYGO) build -skip-dmg
	@set -e; exe=$$(ls -d $(OUT)/*/$(APP).app/Contents/MacOS/$(APP) $(OUT)/*/$(APP) 2>/dev/null | head -1); \
	test -n "$$exe" || { echo "mygo build produced no executable under $(OUT)"; exit 1; }; \
	rm -f bin/$(APP); ln -s "../$$exe" bin/$(APP); echo "built bin/$(APP) -> $$exe"

package: deps-check
	@mkdir -p bin
	$(MYGO) build -skip-dmg
	rm -rf bin/GGUIX-mygo.app
	cp -R $(OUT)/darwin-*/$(APP).app bin/GGUIX-mygo.app

run: build
	./bin/$(APP)

bench: build
	@d=$$(mktemp -d); GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$d GGUIX_BENCH_EXPORT=$$d/bench-export.csv ./bin/$(APP) 2>&1 | grep -E 'GGUIX_METRIC|rror'; true
# gguix:end

# gguix:target web
# MyGo documents no browser or server deployment for web-frontend apps: its
# js/wasm support is for native-UI apps only (see results/mygo/capabilities.md).
web web-remote bench-web:
	@echo "UNSUPPORTED: MyGo has no browser or server mode for web-frontend apps (js/wasm is native UI only)"; exit 3
# gguix:end

# Go tests, Vitest, then Playwright against the test-only HTTP harness
# (main_e2e.go): MyGo has no browser mode to run Playwright against.
test: deps-check
	@mkdir -p bin
	@# Stale result files must not be scored (audit C5).
	rm -f bin/test-go.log bin/test-vitest.json frontend/e2e/results.json
	$(MYGO) generate >/dev/null
	cd frontend && npm run build -s >/dev/null 2>&1
	go build -tags e2e -o bin/$(APP)-e2e .
	go test -v -count=1 ./... 2>&1 | grep -v '^ld:' | tee bin/test-go.log | grep -E '^(--- |FAIL|ok|panic)' || true
	cd frontend && npx vitest run --reporter=json --outputFile=../bin/test-vitest.json >/dev/null 2>&1 || true
	cd frontend && npx playwright test || true
	@for f in bin/test-go.log bin/test-vitest.json frontend/e2e/results.json; do test -s $$f || { echo "missing test result file $$f"; echo "GGUIX_TESTS passed=0 failed=1 not_feasible=0"; exit 1; }; done
	@python3 scripts/gguix_tests.py 0

# MyGo builds with CGO_ENABLED=0 unless the variable is set (the CLI only
# sets it when empty), and GOFLAGS carries the cgoprobe tag through mygo build.
cgoprobe: deps-check
	@mkdir -p bin; \
	if CGO_ENABLED=1 GOFLAGS=-tags=cgoprobe $(MYGO) build -skip-dmg -o bin/probe-out >bin/probe-desktop.log 2>&1; then \
	  if CGO_ENABLED=1 GGUIX_PREFS_DIR=$$(mktemp -d) go test -tags cgoprobe -count=1 -run 'TestExportStreamWriter|TestExportCancelRemovesFile' . >/dev/null 2>&1; \
	  then echo "CGOPROBE desktop PASS builds with CGO_ENABLED=1 and the export code runs with the C checksum"; \
	  else echo "CGOPROBE desktop FAIL builds but the export tests fail"; fi; \
	else echo "CGOPROBE desktop FAIL $$(tail -c 200 bin/probe-desktop.log | tr '\n' ' ')"; fi; \
	echo "CGOPROBE web UNSUPPORTED no web target"; true

cross-windows: deps-check
	$(MYGO) build -skip-dmg -platform windows/amd64

cross-linux: deps-check
	$(MYGO) build -skip-dmg -platform linux/amd64

clean:
	rm -rf bin build frontend/dist .mygo
	rm -f frontend/e2e/results.json frontend/e2e/oracle.json
