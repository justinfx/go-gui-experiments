#!/bin/bash
# System dark set BEFORE launch: does NSGo pick up color-scheme at startup?
. /repo/linux/lib.sh
L=$GGUIX_LOG; O=/repo/results/nsgo/linux
cd /work/apps/nsgo
gsettings set org.gnome.desktop.interface color-scheme prefer-dark; sleep 1
rm -rf /tmp/prefs-gd; mkdir -p /tmp/prefs-gd
GGUIX_PREFS_DIR=/tmp/prefs-gd GGUIX_ROWS=20000 GGUIX_LATENCY_MS=0 ./bin/gguix-nsgo > $L/app-dark-start.log 2>&1 &
APP=$!
W=$(waitwin "GGUIX" 30); sleep 3
shotwin $W $O/x11-dark-at-start.png
kill $APP
