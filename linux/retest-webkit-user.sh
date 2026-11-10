#!/bin/bash
# Re-run Wails (GTK4) and glaze (GTK4) as an unprivileged user, seccomp=unconfined.
# Reuses /work builds from retest-relaxed.sh.
set -u
. /repo/linux/retest-lib.sh 2>/dev/null || true
id tester >/dev/null 2>&1 || useradd -m tester
chmod -R a+rX /work/apps/wails/bin /work/apps/gotth-glaze/bin
for pair in "wails /work/apps/wails/bin/gguix-wails" "gotth-glaze /work/apps/gotth-glaze/bin/gguix-gotth"; do
  set -- $pair; id=$1; bin=$2
  GGUIX_LOG=/repo/results/$id/linux/relaxed-user /repo/linux/run.sh x11 "$id" bash -c '
    . /repo/linux/lib.sh
    mkdir -p /run/user/1001 && chown tester /run/user/1001 && chmod 700 /run/user/1001; su - tester -c "env -i HOME=/home/tester USER=tester PATH=/usr/bin:/bin DISPLAY=$DISPLAY XDG_RUNTIME_DIR=/run/user/1001 GDK_BACKEND=x11 LIBGL_ALWAYS_SOFTWARE=1 WEBKIT_DISABLE_DMABUF_RENDERER=1 dbus-run-session -- '"$bin"'" > $GGUIX_LOG/app.log 2>&1 &
    w=$(waitwin GGUIX 40) || { echo "'"$id"': NO WINDOW"; tail -5 $GGUIX_LOG/app.log; exit 1; }
    sleep 10
    if pgrep -f "'"$bin"'" >/dev/null; then echo "'"$id"': window and alive"; timeout 20 import -window "$w" /repo/results/'"$id"'/linux/relaxed-user-x11-start.png && echo shot-ok; else echo "'"$id"': exited"; fi
    grep -iE "bwrap|sandbox|ERROR" $GGUIX_LOG/app.log | head -4
    pkill -f "'"$bin"'"; true'
done
