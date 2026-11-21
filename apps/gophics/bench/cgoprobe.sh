#!/bin/sh
# Rebuilds each target with -tags cgoprobe and reports PASS/FAIL per target.
cd "$(dirname "$0")/.." || exit 0
mkdir -p bin/cgoprobe
log=bin/cgoprobe/log.txt
: > "$log"

# desktop: build with cgo on, then run the headless export test (Export calls the C checksum).
if CGO_ENABLED=1 go build -tags cgoprobe -o bin/cgoprobe/gguix-gophics . >>"$log" 2>&1 &&
   CGO_ENABLED=1 go test -count=1 -tags cgoprobe -run 'TestT8|TestT1' ./ui >>"$log" 2>&1; then
  echo "CGOPROBE desktop PASS"
else
  echo "CGOPROBE desktop FAIL $(tail -1 "$log")"
fi

# web: wasm has no cgo.
if GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -tags cgoprobe -o bin/cgoprobe/app.wasm . >>"$log" 2>&1; then
  echo "CGOPROBE web PASS"
else
  echo "CGOPROBE web FAIL cgo is unavailable for js/wasm ($(grep -m1 cgoprobe_requires "$log" | sed 's/.*undefined: //'))"
fi

# web-remote: WASM client without cgoprobe, server with it.
if GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -tags remote -o bin/cgoprobe/remote.wasm . >>"$log" 2>&1; then
  echo "CGOPROBE web-remote-client PASS"
else
  echo "CGOPROBE web-remote-client FAIL $(tail -1 "$log")"
fi
if (cd ../../core && CGO_ENABLED=1 go build -tags cgoprobe -o ../apps/gophics/bin/cgoprobe/server ./cmd/server) >>"$log" 2>&1; then
  echo "CGOPROBE web-remote-server PASS"
else
  echo "CGOPROBE web-remote-server FAIL $(tail -1 "$log")"
fi
exit 0
