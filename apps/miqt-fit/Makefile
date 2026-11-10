# GGUIX with MIQT. Run `make deps` once, then any target.
APP     := gguix-miqt
BIN     := bin/$(APP)
LDFLAGS := -s -w
QTBIN   := $(shell brew --prefix qt 2>/dev/null)/bin
DOCKERW := $(CURDIR)/desktop/docker-notty.sh
COREABS := $(abspath ../../core)

.PHONY: deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

# gguix:target desktop
deps:
	@brew list --versions qt >/dev/null 2>&1 || brew install qt
	@brew list --versions pkgconf >/dev/null 2>&1 || brew install pkgconf
	go install github.com/mappu/miqt/cmd/miqt-docker@v0.15.0
	go mod download

# MIQT has no hot reload; an incremental go build is the fastest loop.
dev:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN) . && $(BIN)

build:
	@mkdir -p bin
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN) .

package: build
	@mkdir -p bin
	@test -d bin/GGUIX-miqt.app && mv bin/GGUIX-miqt.app bin/GGUIX-miqt.app.old.$$$$ || true
	mkdir -p bin/GGUIX-miqt.app/Contents/MacOS
	cp $(BIN) bin/GGUIX-miqt.app/Contents/MacOS/$(APP)
	cp desktop/Info.plist bin/GGUIX-miqt.app/Contents/Info.plist
	@echo "size before macdeployqt: $$(du -sh bin/GGUIX-miqt.app | cut -f1)"
	$(QTBIN)/macdeployqt bin/GGUIX-miqt.app
	@echo "size after macdeployqt: $$(du -sh bin/GGUIX-miqt.app | cut -f1)"

run: build
	$(BIN)

cgoprobe:
	@mkdir -p bin
	@if go build -tags cgoprobe -ldflags '$(LDFLAGS)' -o bin/$(APP)-cgoprobe . > bin/cgoprobe.log 2>&1; then \
	  if QT_QPA_PLATFORM=offscreen go test -tags cgoprobe -count=1 -run 'TestT1StartsAndLoads|TestExportCompletes' . >> bin/cgoprobe.log 2>&1; then \
	    echo "CGOPROBE desktop PASS"; \
	  else echo "CGOPROBE desktop FAIL tests failed, see bin/cgoprobe.log"; fi; \
	else echo "CGOPROBE desktop FAIL build error: $$(grep -m1 -i error bin/cgoprobe.log)"; fi

# Needs Docker, miqt-docker (make deps) and the core module mounted at /core.
cross-linux:
	@mkdir -p bin
	DOCKER_DEFAULT_PLATFORM=linux/amd64 DOCKER=$(DOCKERW) MIQTDOCKER_FLAGS="-v $(COREABS):/core:ro" \
	  $(HOME)/go/bin/miqt-docker linux64-go1.27-qt6.12 go build -ldflags '$(LDFLAGS)' -o bin/$(APP)-linux-amd64 .

cross-windows:
	@mkdir -p bin
	DOCKER_DEFAULT_PLATFORM=linux/amd64 DOCKER=$(DOCKERW) MIQTDOCKER_FLAGS="-v $(COREABS):/core:ro -v $(shell go env GOMODCACHE):/gomod -e GOMODCACHE=/gomod -e GOTOOLCHAIN=go1.27.3 -e GOPROXY=off -e GOFLAGS=-mod=mod" \
	  $(HOME)/go/bin/miqt-docker win64-qt6-static -windows-build --tags=windowsqtstatic

clean:
	rm -r -f bin
	rm -f miqt miqt.exe
# gguix:end

# gguix:target web
web web-remote bench-web:
	@echo "UNSUPPORTED: MIQT (Qt Widgets via cgo) has no web target"; exit 3
# gguix:end

# gguix:target test
test:
	@mkdir -p bin
	@rc=0; QT_QPA_PLATFORM=offscreen go test -count=1 -v . > bin/test.log 2>&1 || rc=$$?; \
	grep -v '^GGUIX_TESTS' bin/test.log; \
	line=$$(grep '^GGUIX_TESTS' bin/test.log | tail -1); \
	[ -n "$$line" ] || line="GGUIX_TESTS passed=0 failed=1 not_feasible=0"; \
	echo "$$line"; exit $$rc
# gguix:end

# gguix:target bench
bench: build
	GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$(mktemp -d) GGUIX_BENCH_EXPORT=$$(mktemp -d)/bench.csv $(BIN) 2>&1 | grep GGUIX_METRIC
# gguix:end
