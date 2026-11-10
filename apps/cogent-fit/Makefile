# GGUIX Cogent Core app. See README.md.
CORE      ?= $(HOME)/go/bin/core
BIN        = bin
WEBDIR     = $(BIN)/web
PORT       = 8090
REALGO    := $(shell command -v go)
SHIMDIR    = $(CURDIR)/web
export PATH := $(HOME)/go/bin:$(PATH)

.PHONY: deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean

deps:
	go install cogentcore.org/core@v0.5.27
	go mod download

# gguix:target desktop
dev:
	$(CORE) run

build:
	mkdir -p $(BIN)/_b
	$(CORE) build -o $(BIN)/_b/gguix-cogent
	mv $(BIN)/_b/gguix-cogent/gguix-cogent $(BIN)/gguix-cogent
	rmdir $(BIN)/_b/gguix-cogent $(BIN)/_b

# A stale root GGUIX-cogent.app is removed first and a pack failure fails the
# target, so an old bundle is never moved into bin (audit C4).
package:
	rm -rf GGUIX-cogent.app
	$(CORE) pack -o .
	rm -rf $(BIN)/GGUIX-cogent.app
	mv GGUIX-cogent.app $(BIN)/GGUIX-cogent.app
	rm -f cogent $(notdir $(CURDIR))
	@ls -d $(BIN)/GGUIX-cogent.app

run: build
	./$(BIN)/gguix-cogent
# gguix:end

# gguix:target web
web:
	$(CORE) build wasm -o $(WEBDIR)
	@echo "Serving http://localhost:$(PORT)"
	cd ../../core && go run ./cmd/server -static ../apps/cogent-fit/$(WEBDIR) -addr :$(PORT)

web-remote:
	mkdir -p $(SHIMDIR)/shimbin
	cp $(SHIMDIR)/gotags-shim.sh $(SHIMDIR)/shimbin/go
	GGUIX_REAL_GO=$(REALGO) GGUIX_EXTRA_TAGS=remote PATH=$(SHIMDIR)/shimbin:$$PATH $(CORE) build wasm -o $(WEBDIR)
	@echo "Serving http://localhost:$(PORT) (core server, remote data)"
	cd ../../core && go run ./cmd/server -static ../apps/cogent-fit/$(WEBDIR) -addr :$(PORT)
# gguix:end

# gguix:target test
test:
	@mkdir -p $(BIN)
	@go test -count=1 -v . > $(BIN)/test.log 2>&1; rc=$$?; \
	grep -v -e '^=== ' -e '^PASS$$' -e '^ok ' -e 'GGUIX_TESTS' $(BIN)/test.log; \
	grep 'GGUIX_TESTS' $(BIN)/test.log | tail -1; exit $$rc
# gguix:end

# gguix:target bench
bench: build
	GGUIX_BENCH=1 GGUIX_PREFS_DIR=$$(mktemp -d) GGUIX_BENCH_EXPORT=$$(mktemp -d)/bench-export.csv ./$(BIN)/gguix-cogent

bench-web:
	$(CORE) build wasm -o $(WEBDIR)
	@echo "Open http://localhost:$(PORT)/?bench=1 and read the browser console"
	cd ../../core && go run ./cmd/server -static ../apps/cogent-fit/$(WEBDIR) -addr :$(PORT)
# gguix:end

cgoprobe:
	@mkdir -p $(BIN)
	@if go build -tags cgoprobe -o $(BIN)/cgoprobe-desktop . >$(BIN)/cgoprobe-desktop.log 2>&1; then \
	  if go test -tags cgoprobe -count=1 -run TestT1 . >>$(BIN)/cgoprobe-desktop.log 2>&1; then echo "CGOPROBE desktop PASS builds and T1 runs"; else echo "CGOPROBE desktop FAIL builds but test failed (see $(BIN)/cgoprobe-desktop.log)"; fi; \
	else echo "CGOPROBE desktop FAIL $$(grep -m1 -v '^#' $(BIN)/cgoprobe-desktop.log)"; fi
	@if GOOS=js GOARCH=wasm go build -tags cgoprobe -o $(BIN)/cgoprobe-web.wasm . >$(BIN)/cgoprobe-web.log 2>&1; then echo "CGOPROBE web PASS"; else echo "CGOPROBE web FAIL $$(grep -m1 -v '^#' $(BIN)/cgoprobe-web.log)"; fi
	@if GOOS=js GOARCH=wasm go build -tags remote -o $(BIN)/cgoprobe-web-remote.wasm . >$(BIN)/cgoprobe-wr.log 2>&1; then echo "CGOPROBE web-remote-client PASS (built without cgoprobe)"; else echo "CGOPROBE web-remote-client FAIL $$(grep -m1 -v '^#' $(BIN)/cgoprobe-wr.log)"; fi
	@if (cd ../../core && go build -tags cgoprobe -o ../apps/cogent-fit/$(BIN)/cgoprobe-server ./cmd/server) >$(BIN)/cgoprobe-server.log 2>&1; then echo "CGOPROBE web-remote-server PASS"; else echo "CGOPROBE web-remote-server FAIL $$(grep -m1 -v '^#' $(BIN)/cgoprobe-server.log)"; fi

cross-windows:
	@mkdir -p $(BIN)/windows
	-$(CORE) build -t windows/amd64 -o $(BIN)/windows
	@test -f $(BIN)/windows/gguix-cogent.exe || { echo "core CLI route failed, trying Docker mingw"; \
	docker run --platform linux/amd64 -v $(CURDIR)/../..:/w -v $$(go env GOMODCACHE):/gomod -e GOMODCACHE=/gomod -e GOTOOLCHAIN=local -e GOFLAGS=-mod=mod -e GOOS=windows -e GOARCH=amd64 -e CGO_ENABLED=1 -e CC=x86_64-w64-mingw32-gcc -w /w/apps/cogent-fit golang:1.27.3 \
	sh -c 'apt-get update -qq >/dev/null && apt-get install -y -qq gcc-mingw-w64-x86-64 >/dev/null 2>&1; go build -o bin/windows/gguix-cogent.exe .'; }
	@ls -la $(BIN)/windows/gguix-cogent.exe

cross-linux:
	@mkdir -p $(BIN)/linux
	-$(CORE) build -t linux/amd64 -o $(BIN)/linux
	@test -f $(BIN)/linux/gguix-cogent || { echo "core CLI route failed, trying Docker (about 7 minutes)"; \
	docker run --platform linux/amd64 -v $(CURDIR)/../..:/w -v $$(go env GOMODCACHE):/gomod -e GOMODCACHE=/gomod -e GOTOOLCHAIN=local -e GOFLAGS=-mod=mod -w /w/apps/cogent-fit golang:1.27.3 \
	sh -c 'apt-get update -qq >/dev/null && apt-get install -y -qq gcc libgl1-mesa-dev libegl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev >/dev/null 2>&1; go build -o bin/linux/gguix-cogent .'; }
	@ls -la $(BIN)/linux/gguix-cogent

clean:
	rm -rf bin
	rm -rf web/shimbin
	rm -rf GGUIX-cogent.app
	rm -f cogent $(notdir $(CURDIR))
