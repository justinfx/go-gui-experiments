#!/bin/bash
# Inside the container: sway session, drive the Linux build with virtual keyboard/pointer, capture windows only.
L=/repo/results/iupgo/linux/sway
mkdir -p $L
cd /work/apps/iupgo || exit 1
GGUIX_LOG=$L /repo/linux/run.sh sway iupgo bash -c '
  . /repo/linux/lib.sh
  L='$L'
  cd /work/apps/iupgo
  export GGUIX_PREFS_DIR=/tmp/pf-sway GGUIX_ROWS=3000 GGUIX_LATENCY_MS=0 GGUIX_EXPORT_RPS=1000000
  rm -f /tmp/exp-sway.csv
  ./bin/gguix-iupgo > $L/app.log 2>&1 &
  P=$!
  swwait "GGUIX" 30 > $L/rect.log; cat $L/rect.log
  swaymsg -t get_tree | jq -r ".. | select(.pid? and .app_id?) | \"app_id=\(.app_id) name=\(.name)\"" > $L/appid.log
  swaymsg "[app_id=gguix-iupgo] resize set 1560 900" > /dev/null 2>&1; sleep 3; swshot "GGUIX" $L/start.png
  grep -o "/[^ ]*\.so[^ ]*" /proc/$P/maps | sort -u > $L/maps.log
  # menu bar: click File, capture the window rect plus the popup area below it
  read -r x y w h <<<"$(swrect GGUIX)"
  python3 /repo/linux/wlinput.py click $((x+22)) $((y+10)) left; sleep 1
  grim -g "$x,$y ${w}x$((h>400?400:h))" $L/menu-file.png
  swkey esc; sleep 0.5
  # clipboard: select row 2 then Ctrl+Shift+C
  swclick GGUIX 300 167; sleep 1
  swkey ctrl+shift+c; sleep 1
  wl-paste > $L/clipboard.txt 2>&1; cat $L/clipboard.txt
  swshot "GGUIX" $L/row-selected.png
  swkey ctrl+2; sleep 2; swshot "GGUIX" $L/cards.png
  swkey ctrl+1; sleep 1
  # Export: Ctrl+E opens the GTK save dialog
  swkey ctrl+e; sleep 2
  swaymsg -t get_tree | jq -r ".. | select(.pid? and .app_id?) | \"app_id=\(.app_id) name=\(.name)\"" > $L/dialog-windows.log
  r=$(swrect "Export|Save|Select"); echo "dialog rect: $r" > $L/dialog-rect.log
  [ -n "$r" ] && [ "$r" != "null null null null" ] && swshot "Export|Save|Select" $L/export-dialog.png
  swkey ctrl+a; swtype "/tmp/exp-sway.csv"; sleep 0.5; swkey Return; sleep 4
  ls -la /tmp/exp-sway*  > $L/export-file.log 2>&1
  cp /tmp/exp-sway.csv $L/exp-head.csv 2>/dev/null; head -c 300 $L/exp-head.csv > $L/exp-head.txt; rm -f $L/exp-head.csv
  swshot "GGUIX" $L/after-export.png
  # Host dialog and reveal actions
  swkey ctrl+i; sleep 2
  r=$(swrect "Host Info"); echo "host rect: $r" > $L/host-rect.log
  [ -n "$r" ] && [ "$r" != "null null null null" ] && swshot "Host Info" $L/host-dialog.png
  swclick "Host Info" 125 347; sleep 1
  swclick "Host Info" 305 347; sleep 1
  swshot "Host Info" $L/host-after-reveal.png
  swkey esc; sleep 1
  # system light/dark
  gsettings set org.gnome.desktop.interface color-scheme prefer-dark 2>&1 | head -2
  sleep 3; swshot "GGUIX" $L/system-dark.png
  # quit
  swkey ctrl+q; sleep 2
  if kill -0 $P 2>/dev/null; then echo "ctrl+q: STILL RUNNING" > $L/quit.log; kill $P; else echo "ctrl+q: exited" > $L/quit.log; fi
  cp /tmp/dunst.log $L/dunst-copy.log 2>/dev/null
' > $L/session.log 2>&1
