#!/bin/bash
# Inside the container: x11 session (Xvfb + openbox), drive the Linux build with xdotool, capture windows only.
L=/repo/results/iupgo/linux/x11
mkdir -p $L
cd /work/apps/iupgo || exit 1
GGUIX_LOG=$L /repo/linux/run.sh x11 iupgo bash -c '
  . /repo/linux/lib.sh
  L='$L'
  cd /work/apps/iupgo
  export GGUIX_PREFS_DIR=/tmp/pf-x11 GGUIX_ROWS=3000 GGUIX_LATENCY_MS=0 GGUIX_EXPORT_RPS=1000000
  rm -f /tmp/exp-x11*
  ./bin/gguix-iupgo > $L/app.log 2>&1 &
  P=$!
  W=$(waitwin "GGUIX" 30); echo "window $W" > $L/window.log; sleep 3
  winfo "$W" >> $L/window.log 2>&1
  shotwin "$W" $L/start.png
  grep -o "/[^ ]*\.so[^ ]*" /proc/$P/maps | sort -u > $L/maps.log
  xdotool windowfocus "$W"; sleep 0.5
  # menu: click File, then look for the popup window
  xdotool mousemove --window "$W" 22 10 click 1; sleep 1
  xdotool search --onlyvisible --class "" getwindowname %@ 2>/dev/null | sort -u > $L/windows-with-menu.log
  xdotool key Escape; sleep 0.5
  # select row 2, copy row
  xdotool mousemove --window "$W" 300 167 click 1; sleep 1
  xdotool key ctrl+shift+c; sleep 1
  xclip -o -selection clipboard > $L/clipboard.txt 2>&1; cat $L/clipboard.txt
  shotwin "$W" $L/row-selected.png
  # Cards view
  xdotool key ctrl+2; sleep 2; shotwin "$W" $L/cards.png
  xdotool key ctrl+1; sleep 1
  # Export dialog
  xdotool key ctrl+e; sleep 2
  D=$(waitwin "Export CSV" 10); echo "dialog $D" > $L/dialog.log
  [ -n "$D" ] && [ "$D" != "$W" ] && shotwin "$D" $L/export-dialog.png
  xdotool key ctrl+a; xdotool type --delay 40 "/tmp/exp-x11.csv"; sleep 0.5; xdotool key Return; sleep 4
  ls -la /tmp/exp-x11* > $L/export-file.log 2>&1
  # Host dialog and reveal
  xdotool key ctrl+i; sleep 2
  H=$(waitwin "Host Info" 10); echo "host $H" >> $L/dialog.log
  [ -n "$H" ] && shotwin "$H" $L/host-dialog.png
  xdotool mousemove --window "$H" 115 325 click 1; sleep 1
  xdotool mousemove --window "$H" 275 325 click 1; sleep 1
  xdotool key Escape; sleep 1
  # quit
  xdotool key ctrl+q; sleep 2
  if kill -0 $P 2>/dev/null; then echo "ctrl+q: STILL RUNNING" > $L/quit.log; kill $P; else echo "ctrl+q: exited" > $L/quit.log; fi
' > $L/session.log 2>&1
