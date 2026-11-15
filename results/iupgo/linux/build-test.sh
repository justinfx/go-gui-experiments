#!/bin/bash
# Inside the container: build and test the app from a scratch copy, then start it under Xvfb+openbox.
set -u
/repo/linux/prep.sh iupgo
cd /work/apps/iupgo || exit 1
go build -o bin/gguix-iupgo . 2>&1 | tail -20 > /repo/results/iupgo/linux/make-build.log
ls -la bin/gguix-iupgo >> /repo/results/iupgo/linux/make-build.log 2>&1
ldd bin/gguix-iupgo > /repo/results/iupgo/linux/ldd.log 2>&1
GGUIX_LOG=/repo/results/iupgo/linux/x11 gguix-session x11 iupgo bash -c '
  cd /work/apps/iupgo
  go test -count=1 -v . > /repo/results/iupgo/linux/x11/test.log 2>&1; tail -3 /repo/results/iupgo/linux/x11/test.log
  GGUIX_PREFS_DIR=/tmp/pf ./bin/gguix-iupgo > /repo/results/iupgo/linux/x11/app.log 2>&1 &
  P=$!
  . /repo/linux/lib.sh
  waitwin "GGUIX" 30
  sleep 3
  xdotool search --name "GGUIX" getwindowclassname 2>/dev/null | tee /repo/results/iupgo/linux/x11/wmclass.log
  shotwin "GGUIX" /repo/results/iupgo/linux/x11/start.png
  cp /proc/$P/maps /repo/results/iupgo/linux/x11/maps.raw
  grep -o "/[^ ]*\.so[^ ]*" /proc/$P/maps | sort -u > /repo/results/iupgo/linux/x11/maps.log
  kill $P
' 2>&1 | tail -20
