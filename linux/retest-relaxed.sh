#!/bin/bash
# Inside the container, run with GGUIX_RELAX_SECCOMP=1 (seccomp=unconfined) and
# WITHOUT GGUIX_NO_QT_SANDBOX, so Chromium's sandbox stays on for miqt-gotth.
# Checks the three paths the default container blocks: Wails default GTK4 build,
# glaze GTK4/WebKitGTK 6.0 path, miqt-gotth with the Chromium sandbox enabled
# (run as an unprivileged user, because Chromium refuses its sandbox as root).
# Start + window + window-only screenshot + alive after 8 s.
set -u
/repo/linux/prep.sh wails gotth-glaze miqt-gotth
cd /work/apps/wails && (npm --prefix frontend ci --silent >/dev/null 2>&1 || true) && make build > /repo/results/wails/linux/relaxed-build.log 2>&1; echo "wails build rc=$?"
cd /work/apps/gotth-glaze && make build > /repo/results/gotth-glaze/linux/relaxed-build.log 2>&1; echo "glaze build rc=$?"
cd /work/apps/miqt-gotth && make build > /repo/results/miqt-gotth/linux/relaxed-build.log 2>&1; echo "miqt-gotth build rc=$?"
id tester >/dev/null 2>&1 || useradd -m tester
chmod -R a+rX /work/apps/miqt-gotth/bin

check() { # id binary [user]
  local id=$1 bin=$2 user=${3:-}
  GGUIX_LOG=/repo/results/$id/linux/relaxed /repo/linux/run.sh x11 "$id" bash -c '
    . /repo/linux/lib.sh
    xhost +local: >/dev/null 2>&1 || true
    if [ -n "'"$user"'" ]; then su '"$user"' -c "DISPLAY=$DISPLAY '"$bin"'" > $GGUIX_LOG/app.log 2>&1 &
    else '"$bin"' > $GGUIX_LOG/app.log 2>&1 & fi
    w=$(waitwin GGUIX 40) || { echo "'"$id"': NO WINDOW"; tail -5 $GGUIX_LOG/app.log; exit 1; }
    sleep 8
    if pgrep -f "'"$bin"'" >/dev/null; then echo "'"$id"': window and alive"; else echo "'"$id"': exited"; fi
    shotwin "$w" /repo/results/'"$id"'/linux/relaxed-x11-start.png
    grep -iE "bwrap|sandbox|error" $GGUIX_LOG/app.log | head -5
    pkill -f "'"$bin"'"; true'
}
check wails /work/apps/wails/bin/gguix-wails
check gotth-glaze /work/apps/gotth-glaze/bin/gguix-gotth
check miqt-gotth /work/apps/miqt-gotth/bin/gguix-miqt tester
