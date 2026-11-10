# Helper functions sourced by probe scripts inside a gguix-session. Not a standalone script.
# waitwin <name-regex> [secs]   print window id of first visible window whose NAME matches
waitwin() { local w; for _ in $(seq $((${2:-30}*2))); do w=$(xdotool search --onlyvisible --name "$1" 2>/dev/null | tail -1); [ -n "$w" ] && { echo "$w"; return 0; }; sleep 0.5; done; return 1; }
# shotwin <wid> <out.png>   capture ONE window only (never the root window)
shotwin() { import -window "$1" "$2"; }
# winfo <wid>  geometry, class, name
winfo() { xdotool getwindowgeometry "$1" | tr '\n' ' '; echo; xprop -id "$1" WM_CLASS _NET_WM_NAME _MOTIF_WM_HINTS _GTK_FRAME_EXTENTS 2>/dev/null; }
# dbus_grep <pattern>   matching lines of the dbus-monitor log
dbus_grep() { grep -n -A6 "$1" "$GGUIX_LOG/dbus.log" | head -${2:-40}; }

# ---- Wayland (sway mode) helpers: sway IPC + grim + wtype --------------------
# swrect <app_id-or-title-regex>   prints "x y w h" of the first matching toplevel
swrect() { swaymsg -t get_tree | jq -r --arg p "$1" '[.. | select(.pid? and (.name? // "" | test($p) or ((.app_id? // "") | test($p))))][0].rect | "\(.x) \(.y) \(.width) \(.height)"'; }
# swwait <regex> [secs]  wait for a toplevel; echoes its rect
swwait() { local r; for _ in $(seq $((${2:-30}*2))); do r=$(swrect "$1" 2>/dev/null); [ -n "$r" ] && [ "$r" != "null null null null" ] && { echo "$r"; return 0; }; sleep 0.5; done; return 1; }
# swshot <regex> <out.png>   capture ONLY that window's rectangle with grim
swshot() { local r; read -r x y w h <<<"$(swrect "$1")"; grim -g "$x,$y ${w}x${h}" "$2"; }
# swclick <regex> <x> <y> [left|right]   click at window-relative coordinates (virtual pointer via wlinput)
swclick() { local x y w h; read -r x y w h <<<"$(swrect "$1")"; python3 /repo/linux/wlinput.py click $((x+$2)) $((y+$3)) ${4:-left}; }
# swkey ctrl+shift+c   chord via wlinput (real evdev keycodes, standard us keymap); swtype "text"
swkey() { python3 /repo/linux/wlinput.py key "$1"; }
swtype() { python3 /repo/linux/wlinput.py type "$1"; }
