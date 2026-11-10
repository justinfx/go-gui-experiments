#!/bin/bash
# Cross-compilation for the desktop target. Usage: cross.sh windows|linux
# Both the web server (cmd/web) and the desktop shell (cmd/desktop, glaze with
# purego) are pure Go, so a plain CGO_ENABLED=0 go build is enough: no Docker,
# no mingw, no GTK headers. The result needs WebView2 (Windows) or WebKitGTK
# (Linux, loaded with dlopen at run time) on the target machine.
set -u
os=${1:?windows or linux}
cd "$(dirname "$0")/.."
mkdir -p bin
ext=""; [ "$os" = windows ] && ext=".exe"

for t in web desktop; do
  out="bin/gguix-gotth-glaze-$t-$os-amd64$ext"
  src=./cmd/$t
  flags=""; [ "$os" = windows ] && [ "$t" = desktop ] && flags="-ldflags=-H=windowsgui"
  echo "== $t for $os/amd64 (CGO_ENABLED=0 go build)"
  if CGO_ENABLED=0 GOOS=$os GOARCH=amd64 go build $flags -o "$out" "$src"; then
    echo "CROSS $os $t PASS $(du -h "$out" | cut -f1)"
  else
    echo "CROSS $os $t FAIL (see the error above)"
  fi
done
exit 0
