#!/bin/bash
# Fresh sway session: app_id, shortcuts, clipboard, Host dialog Escape, system light/dark, Ctrl+Q.
. /repo/linux/lib.sh
L=$GGUIX_LOG; O=/repo/results/nsgo/linux
cd /work/apps/nsgo
rm -rf /tmp/prefs-gx; mkdir -p /tmp/prefs-gx
GGUIX_PREFS_DIR=/tmp/prefs-gx GGUIX_ROWS=20000 GGUIX_LATENCY_MS=0 ./bin/gguix-nsgo > $L/app.log 2>&1 &
APP=$!
swwait "GGUIX" 30 | tee $L/rect.txt; sleep 3
swaymsg -t get_tree | jq -r '.. | select(.pid? and .app_id? != null) | "app_id=\(.app_id) name=\(.name) shell=\(.shell)"' > $L/tree.txt; cat $L/tree.txt
swshot "GGUIX" $O/sway-start.png
awk '{print $6}' /proc/$APP/maps | grep '\.so' | sort -u > $O/sway-maps.log
swclick GGUIX 300 200; sleep 1
swkey ctrl+shift+c; sleep 1; wl-paste > $L/clipboard.txt; cat $L/clipboard.txt; echo
swkey ctrl+2; sleep 2; swshot "GGUIX" $O/sway-ctrl2-cards.png; swkey ctrl+1; sleep 1
swkey ctrl+i; sleep 2; swshot "GGUIX" $O/sway-host-dialog.png
swkey esc; sleep 1.5; swshot "GGUIX" $O/sway-host-after-escape.png
gsettings set org.gnome.desktop.interface color-scheme prefer-dark; sleep 3; swshot "GGUIX" $O/sway-system-dark.png
gsettings set org.gnome.desktop.interface color-scheme default; sleep 2
swkey ctrl+q; sleep 2
if kill -0 $APP 2>/dev/null; then echo "ctrl+q: still running" > $L/ctrlq.txt; else echo "ctrl+q: exited" > $L/ctrlq.txt; fi
cat $L/ctrlq.txt
