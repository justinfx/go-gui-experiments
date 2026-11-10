#!/bin/bash
# Cross-compilation attempts for the desktop target. Usage: cross.sh windows|linux
# The web server (cmd/web) is pure Go and cross-compiles directly. The desktop
# shell (cmd/desktop) needs cgo and the platform webview libraries, so it is
# built inside a Docker container that has the C toolchain installed.
set -u
os=${1:?windows or linux}
cd "$(dirname "$0")/.."
mkdir -p bin
ext=""; [ "$os" = windows ] && ext=".exe"

echo "== web server for $os/amd64 (plain go build)"
if CGO_ENABLED=0 GOOS=$os GOARCH=amd64 go build -o "bin/gguix-gotth-web-$os-amd64$ext" ./cmd/web; then
  echo "CROSS $os web PASS $(du -h "bin/gguix-gotth-web-$os-amd64$ext" | cut -f1)"
else
  echo "CROSS $os web FAIL"
fi

echo "== desktop shell for $os/amd64 (Docker golang:1.27.3-bookworm with a C toolchain)"
root=$(cd ../.. && pwd)
if [ "$os" = windows ]; then
  pre='apt-get update -qq && apt-get install -y -qq gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64 >/dev/null && ln -sf /usr/x86_64-w64-mingw32/include/eventtoken.h /usr/x86_64-w64-mingw32/include/EventToken.h'
  env_='GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++'
  flags='-ldflags=-H=windowsgui'
else
  pre='apt-get update -qq && apt-get install -y -qq libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config >/dev/null'
  env_='GOOS=linux GOARCH=amd64 CGO_ENABLED=1'
  flags=''
fi
# Docker runs the amd64 image under emulation on Apple Silicon.
if docker run --rm --platform linux/amd64 -v "$root":/src -w "/src/apps/$(basename "$PWD")" golang:1.27.3-bookworm \
     bash -c "$pre && env $env_ go build $flags -o bin/gguix-gotth-$os-amd64$ext ./cmd/desktop"; then
  echo "CROSS $os desktop PASS $(du -h "bin/gguix-gotth-$os-amd64$ext" | cut -f1)"
else
  echo "CROSS $os desktop FAIL (see the error above)"
fi
exit 0
