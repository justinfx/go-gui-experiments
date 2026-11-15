#!/bin/bash
# Fresh x11 session: window identity, Reveal actions, theme (selector and system), Ctrl+Q.
. /repo/linux/lib.sh
L=$GGUIX_LOG; O=/repo/results/nsgo/linux
cd /work/apps/nsgo
rm -rf /tmp/prefs-gx /tmp/export-out; mkdir -p /tmp/prefs-gx /tmp/export-out
GGUIX_PREFS_DIR=/tmp/prefs-gx GGUIX_EXPORT_RPS=-1 GGUIX_ROWS=20000 GGUIX_LATENCY_MS=0 ./bin/gguix-nsgo > $L/app.log 2>&1 &
APP=$!
W=$(waitwin "GGUIX" 30); sleep 3
xdotool windowfocus $W; sleep .5
C() { xdotool mousemove --window $W $1 $2 click 1; sleep ${3:-1}; }
winfo $W > $L/winfo.txt; cat $L/winfo.txt
C 300 200
# Export once so Reveal Last Export is enabled (save dialog through the portal)
xdotool key ctrl+e; sleep 5
D=$(xdotool search --onlyvisible --name "Save" | tail -1); xdotool windowfocus $D; sleep 1; xdotool key ctrl+a; xdotool type --delay 150 "/tmp/export-out/out.csv"; sleep 1; xdotool key Delete; xdotool key Return; sleep 6
xdotool windowfocus $W; sleep .5
xdotool key ctrl+i; sleep 2; shotwin $W $O/x11-host-dialog2.png
# Reveal buttons: positions come from x11-host-dialog2.png (found by hand): config at 546,574; last export at 746,574
C 546 574 3; C 746 574 3
shotwin $W $O/x11-host-after-reveal.png
cp $L/xdg-open.log $L/xdg-open-copy.txt 2>/dev/null
grep -c "OpenURI" $L/dbus.log > $L/openuri-count.txt
C 802 610 1
# Theme: Dark from the toolbar select
C 1213 50 1; shotwin $W $O/x11-theme-menu.png
C 1126 105 2; shotwin $W $O/x11-theme-dark.png; sleep 1; cp /tmp/prefs-gx/prefs.json $L/prefs-after-dark.json
# System, then flip the OS setting
C 1213 50 1; C 1126 53 2
PS() { gdbus call --session --dest org.freedesktop.portal.Desktop --object-path /org/freedesktop/portal/desktop --method org.freedesktop.portal.Settings.Read org.freedesktop.appearance color-scheme 2>&1; }
PS > $L/portal-before.txt
gsettings get org.gnome.desktop.interface color-scheme > $L/gsettings-before.txt
gsettings set org.gnome.desktop.interface color-scheme prefer-dark; sleep 3
gsettings get org.gnome.desktop.interface color-scheme > $L/gsettings-after.txt
shotwin $W $O/x11-system-dark.png
PS > $L/portal-after.txt; cat $L/portal-before.txt $L/portal-after.txt
xdotool windowfocus $W; sleep .5; xdotool key ctrl+q; sleep 2
if kill -0 $APP 2>/dev/null; then echo "ctrl+q: still running" > $L/ctrlq.txt; else echo "ctrl+q: exited" > $L/ctrlq.txt; fi
cat $L/ctrlq.txt
