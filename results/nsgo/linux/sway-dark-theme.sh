#!/bin/bash
# What the Ubuntu dark toggle writes (color-scheme + a -dark gtk-theme), set before launch and live.
. /repo/linux/lib.sh
L=$GGUIX_LOG; O=/repo/results/nsgo/linux
cd /work/apps/nsgo
pkg-config --modversion gtk4 > $L/gtk4-version.txt; cat $L/gtk4-version.txt
gsettings set org.gnome.desktop.interface color-scheme prefer-dark
gsettings set org.gnome.desktop.interface gtk-theme Adwaita-dark; sleep 1
rm -rf /tmp/prefs-gd; mkdir -p /tmp/prefs-gd
GGUIX_PREFS_DIR=/tmp/prefs-gd GGUIX_ROWS=20000 GGUIX_LATENCY_MS=0 ./bin/gguix-nsgo > $L/app-dark-theme.log 2>&1 &
APP=$!
swwait "GGUIX" 30 >/dev/null; sleep 3
swshot "GGUIX" $O/sway-dark-theme-at-start.png
gsettings set org.gnome.desktop.interface color-scheme default
gsettings set org.gnome.desktop.interface gtk-theme Adwaita; sleep 3
swshot "GGUIX" $O/sway-light-theme-live.png
kill $APP
