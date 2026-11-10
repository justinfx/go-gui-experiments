#!/bin/sh
# Host side: run a command in the GGUIX Linux container (native linux/arm64).
#   linux/dx.sh build            build the image
#   linux/dx.sh <cmd...>         run cmd in a throwaway container (e.g. bash, or gguix-session x11 fyne ...)
# Mounts: repo at /repo (rw), ~/go/pkg/mod at /gomod (rw, GOMODCACHE), Linux-only volumes gguix-gopath (/root/go, go install tools),
# gguix-gocache (/gocache = GOCACHE) and gguix-work (/work = per-run copy of apps/ and core/).
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
if [ "${1:-}" = build ]; then
  exec docker build --platform linux/arm64 -t gguix-linux -f "$root/linux/Dockerfile" "$root/linux"
fi
# Default: NO sandbox relaxation. Two explicit opt-ins exist for apps whose web engine cannot start in a root,
# default-seccomp container; they weaken the container's isolation of the app under test, so use them knowingly:
#   GGUIX_NO_QT_SANDBOX=1    sets QTWEBENGINE_DISABLE_SANDBOX=1 (Qt WebEngine refuses to run as root otherwise)
#   GGUIX_RELAX_SECCOMP=1    adds --security-opt seccomp=unconfined (lets WebKitGTK 6.0 / bwrap create user namespaces)
extra=""
[ "${GGUIX_NO_QT_SANDBOX:-}" = 1 ] && extra="$extra -e QTWEBENGINE_DISABLE_SANDBOX=1"
[ "${GGUIX_RELAX_SECCOMP:-}" = 1 ] && extra="$extra --security-opt seccomp=unconfined --security-opt systempaths=unconfined"
# shellcheck disable=SC2086
exec docker run --rm -i --platform linux/arm64 --shm-size=1g $extra \
  -v "$root":/repo -v "$HOME/go/pkg/mod":/gomod \
  -v gguix-gocache:/gocache -v gguix-work:/work -v gguix-gopath:/root/go \
  gguix-linux "$@"
