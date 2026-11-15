#!/bin/bash
/repo/linux/prep.sh iupgo
cd /work/apps/iupgo && go build -o bin/gguix-iupgo . || exit 1
/repo/linux/run.sh x11 iupgo bash -c '
  cd /work/apps/iupgo
  GGUIX_DEBUG_IMG=1 GGUIX_PREFS_DIR=/tmp/pf timeout 8 ./bin/gguix-iupgo 2>&1 | head -20
' 2>&1 | tail -20
