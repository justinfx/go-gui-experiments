#!/bin/bash
# miqt-docker passes -t when it thinks a tty exists; drop it so this works from make/CI.
out=()
for a in "$@"; do
  case "$a" in -t) ;; -it|-ti) out+=(-i);; *) out+=("$a");; esac
done
exec docker "${out[@]}"
