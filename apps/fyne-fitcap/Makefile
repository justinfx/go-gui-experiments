# GGUIX Fyne app. Run from apps/fyne-fitcap.
GOBIN := $(shell go env GOPATH)/bin
FYNE  := $(GOBIN)/fyne
FYNE_CLI_VERSION := v1.7.4
BIN   := bin/gguix-fyne
SERVE_PORT := 8090

.PHONY: deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

deps:
	go install fyne.io/tools/cmd/fyne@$(FYNE_CLI_VERSION)
	go mod download

dev:
	@echo "Fyne has no hot reload; this is go run (restart to pick up edits)."
	go run .

build:
	mkdir -p bin
	go build -tags release -trimpath -ldflags "-s -w" -o $(BIN) .

package:
	mkdir -p bin
	$(FYNE) package --release --target darwin --name GGUIX-fyne --app-id io.gguix.fyne --icon Icon.png --source-dir .
	rm -r -f bin/GGUIX-fyne.app
	mv GGUIX-fyne.app bin/GGUIX-fyne.app

run: build
	./$(BIN)

# gguix:target web
# fyne package --target wasm writes wasm/ next to the sources; it is moved to bin/web.
define build_web
rm -r -f wasm bin/web
$(FYNE) package --release --target wasm --name gguix-fyne --icon Icon.png --source-dir . $(1)
mkdir -p bin
mv wasm bin/web
endef

web:
	$(call build_web,)
	cd ../../core && go run ./cmd/server -static ../apps/fyne-fitcap/bin/web -addr :$(SERVE_PORT)

web-remote:
	$(call build_web,--tags remote)
	cd ../../core && go run ./cmd/server -static ../apps/fyne-fitcap/bin/web -addr :$(SERVE_PORT)

bench-web:
	$(call build_web,)
	@echo "Open http://localhost:$(SERVE_PORT)/?bench=1 and watch the browser console"
	cd ../../core && go run ./cmd/server -static ../apps/fyne-fitcap/bin/web -addr :$(SERVE_PORT)
# gguix:end

test:
	@out=$$(mktemp); go test -count=1 -v ./... > $$out 2>&1; st=$$?; \
	grep -v "duplicate libraries" $$out | grep -v '^=== ' ; \
	line=$$(grep '^GGUIX_TESTS ' $$out | tail -1); \
	[ -z "$$line" ] && line="GGUIX_TESTS passed=0 failed=1 not_feasible=0"; \
	rm -f $$out; echo "$$line"; exit $$st

bench: build
	GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$(mktemp -d) GGUIX_BENCH_EXPORT=$$(mktemp -u).csv ./$(BIN)

cgoprobe:
	@mkdir -p bin
	@if go build -tags cgoprobe -o bin/gguix-fyne-cgo . 2>bin/cgo-desktop.log; then echo "CGOPROBE desktop PASS"; \
	  else echo "CGOPROBE desktop FAIL $$(grep -v 'duplicate libraries' bin/cgo-desktop.log | head -1)"; fi
	@if GOOS=js GOARCH=wasm go build -tags cgoprobe -o bin/cgo.wasm . 2>bin/cgo-web.log; then echo "CGOPROBE web PASS"; \
	  else echo "CGOPROBE web FAIL $$(sed -n 2p bin/cgo-web.log)"; fi
	@if GOOS=js GOARCH=wasm go build -tags remote -o bin/remote.wasm . 2>bin/cgo-wr.log; then echo "CGOPROBE web-remote-client PASS"; \
	  else echo "CGOPROBE web-remote-client FAIL $$(head -1 bin/cgo-wr.log)"; fi
	@if (cd ../../core && go build -tags cgoprobe -o ../apps/fyne-fitcap/bin/server-cgo ./cmd/server) 2>bin/cgo-srv.log; then echo "CGOPROBE web-remote-server PASS"; \
	  else echo "CGOPROBE web-remote-server FAIL $$(head -1 bin/cgo-srv.log)"; fi

cross-windows:
	@which docker >/dev/null || { echo "UNSUPPORTED: docker missing"; exit 3; }
	go install github.com/fyne-io/fyne-cross@latest
	go mod vendor
	$(GOBIN)/fyne-cross windows -arch=amd64 -app-id io.gguix.fyne -name gguix-fyne -env GOTOOLCHAIN=auto -env GOFLAGS=-mod=vendor ; st=$$?; rm -r vendor; exit $$st

cross-linux:
	@which docker >/dev/null || { echo "UNSUPPORTED: docker missing"; exit 3; }
	go install github.com/fyne-io/fyne-cross@latest
	go mod vendor
	$(GOBIN)/fyne-cross linux -arch=amd64 -app-id io.gguix.fyne -name gguix-fyne -env GOTOOLCHAIN=auto -env GOFLAGS=-mod=vendor ; st=$$?; rm -r vendor; exit $$st

clean:
	rm -r -f bin wasm fyne-cross vendor GGUIX-fyne.app
