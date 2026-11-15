#!/bin/bash
# Runs INSIDE a gguix-session x11 nsgo (image with xkbcli, gnome-themes-extra and the portals; nothing installed ad hoc).
# The session's run.sh does not start a portal; xdg-desktop-portal is D-Bus activated on first use. XDG_CURRENT_DESKTOP is GNOME
# and there is no gnome-shell, so the portal config below prefers the gtk backend (only the session's own XDG_CONFIG_HOME is written).
. /repo/linux/lib.sh
L=$GGUIX_LOG; O=/repo/results/nsgo/linux
cd /work/apps/nsgo
rm -rf /tmp/prefs-gx /tmp/export-out; mkdir -p /tmp/prefs-gx /tmp/export-out
printf 'hello notes from a text file\n' > /tmp/notes.txt
GGUIX_PREFS_DIR=/tmp/prefs-gx GGUIX_EXPORT_RPS=-1 GGUIX_ROWS=20000 GGUIX_LATENCY_MS=0 ./bin/gguix-nsgo > $L/app.log 2>&1 &
APP=$!
W=$(waitwin "GGUIX" 30); sleep 3
xdotool windowfocus $W; sleep .5
C() { xdotool mousemove --window $W $1 $2 click 1; sleep ${3:-1}; }
wins() { xdotool search --onlyvisible --name "." 2>/dev/null | while read w; do n=$(xdotool getwindowname $w 2>/dev/null); [ -n "$n" ] && echo "$w $n"; done; }
dialog() { wins | grep -v -E "GGUIX - NSGo|Openbox" | head -1 | cut -d' ' -f1; }
winfo $W > $L/winfo.txt; cat $L/winfo.txt
shotwin $W $O/x11-start.png
awk '{print $6}' /proc/$APP/maps | grep '\.so' | sort -u > $O/x11-maps.log
C 300 200
xdotool key ctrl+shift+c; sleep 1; xclip -o -selection clipboard > $L/clipboard.txt; cat $L/clipboard.txt
xdotool key ctrl+2; sleep 2; shotwin $W $O/x11-ctrl2-cards.png; xdotool key ctrl+1; sleep 1
# Host dialog, then Escape
xdotool key ctrl+i; sleep 1.5; shotwin $W $O/x11-host-dialog.png
xdotool key Escape; sleep 1.5; shotwin $W $O/x11-host-after-escape.png
# Import Notes through the portal dialog
xdotool key ctrl+o; sleep 5; D=$(dialog); echo "import dialog: $D"
shotwin $D $O/x11-import-dialog.png; xprop -id $D WM_CLASS WM_TRANSIENT_FOR _NET_WM_WINDOW_TYPE > $L/import-dialog-props.txt
xdotool windowfocus $D; sleep 1; xdotool key ctrl+l; sleep 1; xdotool type --delay 150 "/tmp/notes.txt"; sleep 1; xdotool key Delete; sleep .5; shotwin $D $O/x11-import-dialog-typed.png; xdotool key Return; sleep 3
shotwin $W $O/x11-after-import.png
# Export through the portal dialog
xdotool windowfocus $W; sleep .5; xdotool key ctrl+e; sleep 5; D=$(dialog); echo "export dialog: $D"
shotwin $D $O/x11-export-dialog.png; xprop -id $D WM_CLASS WM_TRANSIENT_FOR _NET_WM_WINDOW_TYPE > $L/export-dialog-props.txt
xdotool windowfocus $D; sleep 1; xdotool key ctrl+a; xdotool type --delay 150 "/tmp/export-out/out.csv"; sleep .5; xdotool key Return; sleep 6
ls -la /tmp/export-out > $L/export-files.txt; cat $L/export-files.txt
grep -B2 -A8 "interface=org.freedesktop.Notifications; member=Notify" $L/dbus.log | head -16 > $L/notify.txt
shotwin $W $O/x11-after-export.png
# Host dialog reveal actions
xdotool windowfocus $W; sleep .5; xdotool key ctrl+i; sleep 1.5; shotwin $W $O/x11-host-after-export.png
