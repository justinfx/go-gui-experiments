#!/bin/sh
# Measure one app (see bench/measure.py). Run sequentially, nothing else running.
#   bench/measure.sh <id> [--runs 5] [--skip-cross] [--only step,step]
set -e
cd "$(dirname "$0")/.."
[ -d bench/node_modules/playwright ] || (cd bench && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm install --no-audit --no-fund)
exec python3 -I bench/measure.py "$@"
