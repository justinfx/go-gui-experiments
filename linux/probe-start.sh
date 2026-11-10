#!/bin/bash
# Run inside a gguix-session: start a binary, wait, report whether the process is still alive, then kill it.
# Usage (inside session): probe-start.sh <binary> [seconds]
# Used for the weston-headless column where no input injection or screenshot is possible.
bin=$1; secs=${2:-12}
export GGUIX_PREFS_DIR=$(mktemp -d /tmp/prefs.XXXXXX)
( "$bin" > "$GGUIX_LOG/app.log" 2>&1 & )
sleep "$secs"
if pgrep -f "$bin" > /dev/null; then echo "process alive after ${secs}s"; else echo "process DEAD"; fi
grep -v '^\s*$' "$GGUIX_LOG/app.log" | grep -v "has no notify" | head -6 | cut -c1-200
pkill -9 -f "$bin"
