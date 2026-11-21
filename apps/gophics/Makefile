# GGUIX reference app on Gophics (github.com/doug/gophics @ 8d36097).
# Every Go build is CGO_ENABLED=0 except the cgoprobe target.

GOPHICS_VERSION := v0.0.0-20261103180631-8d360971eb89
TOOLS      := $(CURDIR)/.tools
GOPHICS    := $(TOOLS)/gophics
SERVE_PORT := 8090
BIN        := bin
APP        := $(BIN)/gguix-gophics
BENCH_DIR  := $(CURDIR)/bin/bench-run
export CGO_ENABLED := 0

.PHONY: deps dev build package run web web-remote test bench bench-web cgoprobe cross-windows cross-linux clean serve

deps:
	@test -x $(GOPHICS) || GOBIN=$(TOOLS) go install github.com/doug/gophics/cmd/gophics@$(GOPHICS_VERSION)
	@go mod download

# gguix:target desktop
dev: deps
	$(GOPHICS) dev -p desktop .

build: deps
	$(GOPHICS) build -p desktop -o $(BIN)/desktop .
	rm -f $(APP) && cp $(BIN)/desktop/app $(APP)

package: deps
	$(GOPHICS) package -name GGUIX-gophics -id com.gguix.gophics -dmg=false -o $(BIN)/package .
	rm -rf $(BIN)/GGUIX-gophics.app
	mv $(BIN)/package/GGUIX-gophics.app $(BIN)/GGUIX-gophics.app

run: build
	./$(APP)

bench: build
	mkdir -p $(BENCH_DIR)
	GGUIX_BENCH=1 GGUIX_PREFS_DIR=$(BENCH_DIR) GGUIX_BENCH_EXPORT=$(BENCH_DIR)/bench-export.csv ./$(APP)

cross-windows: deps
	GOOS=windows GOARCH=amd64 go build -o $(BIN)/cross/gguix-gophics.exe .
	@ls -l $(BIN)/cross/gguix-gophics.exe

cross-linux: deps
	GOOS=linux GOARCH=amd64 go build -o $(BIN)/cross/gguix-gophics-linux-amd64 .
	@ls -l $(BIN)/cross/gguix-gophics-linux-amd64
# gguix:end

# gguix:target web
web: deps
	$(GOPHICS) build -p web -o $(BIN)/web .
	$(MAKE) serve

web-remote: deps
	$(GOPHICS) build -p web -tags remote -o $(BIN)/web .
	$(MAKE) serve

bench-web: web

serve:
	cd ../../core && go run ./cmd/server -static ../apps/gophics/bin/web -addr :$(SERVE_PORT)
# gguix:end

# gguix:target bench
cgoprobe: deps
	@./bench/cgoprobe.sh
# gguix:end

test: deps
	@./run_tests.sh

clean:
	rm -rf $(BIN)
