#!/bin/bash
# Inside a gguix-session (x11 or sway): file dialogs (zenity), export notification, Reveal,
# system light/dark, clipboard. Build first (see check-x11.sh header), then:
#   /repo/linux/run.sh <x11|sway> gophics /repo/results/gophics/linux/check-full.sh
. /repo/linux/lib.sh
M=$GGUIX_MODE; L=$GGUIX_LOG; O=/repo/results/gophics/linux/$M
mkdir -p "$O"; cd /work/apps/gophics
rm -rf /tmp/prefs-gp /tmp/export-out; mkdir -p /tmp/prefs-gp /tmp/export-out
echo "hello notes from a text file" > /tmp/notes.txt

if [ "$M" = x11 ]; then
  K() { xdotool key --clearmodifiers "$1"; }
  T() { printf %s "$1" | xclip -selection clipboard; xdotool key --clearmodifiers ctrl+v; }  # paste: typed input races the chooser's completion
  CL() { xdotool mousemove --window "$W" "$1" "$2" click 1; }
  SH() { shotwin "$W" "$O/$1"; MASK "$O/$1"; }
  ZW() { for _ in $(seq 30); do Z=$(xdotool search --onlyvisible --class zenity 2>/dev/null | tail -1); [ -n "$Z" ] && return 0; sleep 0.5; done; return 1; }
  ZSH() { shotwin "$Z" "$O/$1"; xprop -id "$Z" WM_CLASS WM_NAME WM_TRANSIENT_FOR _NET_WM_WINDOW_TYPE > "$O/${1%.png}-props.log" 2>&1; }
  WAITAPP() { W=$(waitwin "GGUIX" 30); xdotool windowactivate "$W" 2>/dev/null; }
  ZFOCUS() { xdotool windowactivate "$Z"; sleep 0.5; }
  AF() { xdotool windowactivate "$W"; sleep 0.5; }
  KEYESC=Escape; KEYRET=Return; PASTE() { xclip -o -selection clipboard; }
else
  K() { swkey "$1"; }
  T() { swtype "$1"; }
  CL() { swclick GGUIX "$1" "$2"; }
  SH() { swshot GGUIX "$O/$1"; MASK "$O/$1"; }
  ZW() { swwait zenity 15 >/dev/null; }
  ZSH() { swshot zenity "$O/$1"; swaymsg -t get_tree | jq -r '.. | select(.pid? and .app_id? != null) | "app_id=\(.app_id) name=\(.name) type=\(.type)"' > "$O/${1%.png}-props.log"; }
  WAITAPP() { swwait GGUIX 30 >/dev/null; }
  ZFOCUS() { :; }
  AF() { :; }
  KEYESC=esc; KEYRET=return; PASTE() { wl-paste; }
fi
# Blank the status bar's user@host label (container hostname) in app captures.
MASK() { local w h; read -r w h <<<"$(identify -format '%w %h' "$1")"; convert "$1" -fill '#888' -draw "rectangle $((w-240)),$((h-34)) $((w-75)),$((h-6))" "$1"; }
run() { GGUIX_ROWS=1000 GGUIX_PREFS_DIR=/tmp/prefs-gp "$@" ./bin/gguix-gophics >> "$L/app-full.log" 2>&1 & APP=$!; WAITAPP; sleep 4; }

# 1. Start with the portal saying dark (color-scheme prefer-dark), as NSGo was checked.
gsettings set org.gnome.desktop.interface color-scheme prefer-dark; sleep 1
gdbus call --session -d org.freedesktop.portal.Desktop -o /org/freedesktop/portal/desktop \
  -m org.freedesktop.portal.Settings.ReadOne org.freedesktop.appearance color-scheme > "$O/portal-dark.log" 2>&1
run env; SH dark-portal-at-start.png; kill $APP; wait $APP 2>/dev/null
gsettings set org.gnome.desktop.interface color-scheme default; sleep 1

# 2. Start with GTK_THEME=Adwaita:dark (what Gophics' detectDarkMode reads).
run env GTK_THEME=Adwaita:dark; SH dark-gtktheme-at-start.png; kill $APP; wait $APP 2>/dev/null

# 3. Main run: live switch, clipboard, Import (open dialog), Export (folder chooser), notify, Reveal.
run env
CL 300 300; sleep 1
gsettings set org.gnome.desktop.interface color-scheme prefer-dark; sleep 3; SH live-dark.png
gsettings set org.gnome.desktop.interface color-scheme default; sleep 1
K ctrl+shift+c; sleep 1; PASTE > "$O/clipboard-full.log" 2>&1

K ctrl+o; if ZW; then sleep 2; ZFOCUS; ZSH import-dialog.png; K ctrl+l; sleep 0.5; T /tmp/notes.txt; sleep 1; ZSH import-dialog-typed.png; K $KEYRET; sleep 3; SH after-import.png
else echo "no zenity window" > "$O/import-dialog-props.log"; fi

AF; K ctrl+e; sleep 2; SH export-name-dialog.png; K $KEYRET
if ZW; then sleep 2; ZFOCUS; ZSH export-folder-dialog.png; K ctrl+l; sleep 0.5; T /tmp/export-out; sleep 1; ZSH export-folder-typed.png; K $KEYRET; sleep 6; SH after-export.png
else echo "no zenity window" > "$O/export-folder-dialog-props.log"; fi
ls -l /tmp/export-out > "$O/export-files.log" 2>&1; wc -l /tmp/export-out/*.csv >> "$O/export-files.log" 2>&1

AF; CL 1238 780; sleep 2; SH host-reveal.png  # Host button
CL 768 526; sleep 1.5; CL 598 526; sleep 1.5
K $KEYESC; sleep 1
cp "$L/xdg-open.log" "$O/xdg-open-full.log" 2>/dev/null || echo "no xdg-open calls" > "$O/xdg-open-full.log"
grep -n -A8 'Notifications; member=Notify' "$L/dbus.log" > "$O/notify.log" 2>&1
cp "$L/dunst.log" "$O/dunst-full.log" 2>/dev/null
kill $APP; wait $APP 2>/dev/null
exit 0
