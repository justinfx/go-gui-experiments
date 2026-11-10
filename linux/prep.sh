#!/bin/bash
# Inside the container: copy apps/<id> and core/ from /repo into /work so Linux builds
# never overwrite the macOS binaries in apps/<id>/bin or node_modules. Usage: prep.sh <id>...
set -eu
mkdir -p /work/apps
rsync -a --delete --exclude bin --exclude node_modules --exclude '*.app' --exclude fyne-cross --exclude test-results /repo/core/ /work/core/
for id in "$@"; do
  # keep a previous Linux node_modules / bin across re-preps
  rsync -a --delete --exclude bin --exclude node_modules --exclude '*.app' --exclude fyne-cross --exclude test-results --exclude 'frontend/dist' \
    /repo/apps/$id/ /work/apps/$id/
done
