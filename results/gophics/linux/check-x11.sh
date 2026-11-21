#!/bin/bash
# Inside a gguix-session (x11 or sway): start the Linux build, capture window, libs, keys.
# Usage (host): linux/dx.sh bash -c '/repo/linux/prep.sh gophics && cd /work/apps/gophics && \
#   CGO_ENABLED=0 go build -o bin/gguix-gophics . && /repo/linux/run.sh x11 gophics /repo/results/gophics/linux/check-x11.sh'
. /repo/linux/lib.sh
out=/repo/results/gophics/linux/$GGUIX_MODE
mkdir -p "$out" /tmp/prefs-gp
cd /work/apps/gophics
ldd bin/gguix-gophics > "$out/ldd.log" 2>&1
file bin/gguix-gophics > "$out/file.log"
GGUIX_ROWS=1000 GGUIX_PREFS_DIR=/tmp/prefs-gp GOPHICS_GPU_LOG=info ./bin/gguix-gophics > "$out/app.log" 2>&1 &
pid=$!
sleep 12
if ! kill -0 $pid 2>/dev/null; then echo "app exited" >> "$out/app.log"; exit 0; fi
grep -v '^$' /proc/$pid/maps | awk '{print $6}' | grep '\.so' | sort -u > "$out/maps.log"
if [ "$GGUIX_MODE" = x11 ]; then
  w=$(waitwin "GGUIX" 20)
  winfo "$w" > "$out/window.log" 2>&1
  shotwin "$w" "$out/start.png"
  xdotool windowactivate "$w" 2>/dev/null; xdotool mousemove --window "$w" 300 300 click 1; sleep 1
  xdotool key ctrl+2; sleep 2; shotwin "$w" "$out/ctrl2-cards.png"
  xdotool key ctrl+shift+c; sleep 1; xclip -o -selection clipboard > "$out/clipboard.log" 2>&1
  xdotool key ctrl+i; sleep 2; shotwin "$w" "$out/host-dialog.png"
  xdotool key Escape; sleep 1
  xdotool key ctrl+q; sleep 3
  if kill -0 $pid 2>/dev/null; then echo "ctrl+q: still running" > "$out/ctrlq.log"; kill $pid; else echo "ctrl+q: exited" > "$out/ctrlq.log"; fi
else
  swaymsg -t get_tree > "$out/tree.json" 2>&1
  swshot "GGUIX" "$out/start.png" 2>>"$out/app.log"
  kill $pid
fi
wait $pid 2>/dev/null
exit 0
