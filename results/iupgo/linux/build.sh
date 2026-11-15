#!/bin/bash
# Inside the container: scratch copy, build, ldd, themes, and the tests with and without a dark GTK theme.
/repo/linux/prep.sh iupgo
cd /work/apps/iupgo || exit 1
go build -tags ctrl,matrix -o bin/gguix-iupgo . && go build -o bin/gguix-iupgo-native . > /repo/results/iupgo/linux/make-build.log 2>&1
ls -la bin/gguix-iupgo >> /repo/results/iupgo/linux/make-build.log
ldd bin/gguix-iupgo > /repo/results/iupgo/linux/ldd.log 2>&1
ls /usr/share/themes; which xkbcli; dpkg -l | grep -c gnome-themes-extra > /repo/results/iupgo/linux/gtk-themes.log 2>&1
for t in ""; do
  GGUIX_LOG=/tmp/x11-test /repo/linux/run.sh x11 iupgo bash -c 'cd /work/apps/iupgo; go test -tags ctrl,matrix -count=1 -v .; echo NATIVE; go test -count=1 -v . 2>&1 | grep -E "^(--- |FAIL|ok|\s+app_test)"' > "/repo/results/iupgo/linux/test-theme-${t:-default}.log" 2>&1
done
