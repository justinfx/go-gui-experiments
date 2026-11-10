#!/bin/sh
# Wraps `go` so that `core build wasm` (which hardcodes -tags mvmnosrc and has
# no tags option) also gets the tags in $GGUIX_EXTRA_TAGS.
real="$GGUIX_REAL_GO"
if [ "$1" = "build" ] && [ -n "$GGUIX_EXTRA_TAGS" ]; then
  n=$#
  i=0
  prev=""
  while [ $i -lt $n ]; do
    a="$1"
    shift
    if [ "$prev" = "-tags" ]; then a="$a,$GGUIX_EXTRA_TAGS"; fi
    set -- "$@" "$a"
    prev="$1"
    prev="$a"
    i=$((i + 1))
  done
  exec "$real" "$@"
fi
exec "$real" "$@"
