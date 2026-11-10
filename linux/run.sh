#!/bin/bash
# GGUIX Linux session launcher (runs INSIDE the container).
#
#   gguix-session <x11|wayland|sway> <app-id> [command args...]
#
# Starts a private session: dbus session bus, a display server, a window manager
# (X11 only), dunst (org.freedesktop.Notifications) and a dbus-monitor log.
# Then runs the command (default: an interactive bash) inside it and tears
# everything down when the command exits.
#
#   x11     Xvfb :N + openbox. xdotool/xclip work.
#   wayland weston --backend=headless (pixman). No XWayland, so GDK_BACKEND=wayland
#           and QT_QPA_PLATFORM=wayland clients really use the Wayland protocol.
#           weston has no virtual-keyboard protocol, so input cannot be injected;
#           use 'sway' for that.
#   sway    sway on the headless wlroots backend (grim, wtype, wl-clipboard work).
#
# Logs go to $GGUIX_LOG (default /repo/results/<app-id>/linux/<mode>): dbus.log, display.log,
# dunst.log. Exported to the command: DISPLAY / WAYLAND_DISPLAY, GDK_BACKEND,
# QT_QPA_PLATFORM, DBUS_SESSION_BUS_ADDRESS, GGUIX_LOG, GGUIX_MODE.
set -u
mode=${1:?usage: gguix-session <x11|wayland|sway> <app-id> [cmd...]}
app=${2:?app id}
shift 2

export GGUIX_MODE=$mode
export GGUIX_LOG=${GGUIX_LOG:-/repo/results/$app/linux/$mode}
mkdir -p "$GGUIX_LOG"
export XDG_RUNTIME_DIR=/run/user/0
mkdir -p "$XDG_RUNTIME_DIR" && chmod 700 "$XDG_RUNTIME_DIR"
export XDG_CONFIG_HOME=$(mktemp -d /tmp/xdgc.XXXXXX) XDG_DATA_HOME=$(mktemp -d /tmp/xdgd.XXXXXX)
export XDG_CURRENT_DESKTOP=${XDG_CURRENT_DESKTOP:-GNOME}
export LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe
unset DISPLAY WAYLAND_DISPLAY
export PATH=/repo/linux/shim:$PATH   # logging xdg-open stand-in

pids=()
cleanup() { for p in "${pids[@]:-}"; do [ -n "$p" ] && kill "$p" 2>/dev/null; done; wait 2>/dev/null; }
trap cleanup EXIT

# Session bus (kept alive for the whole session; address exported).
eval "$(dbus-launch --sh-syntax)"
export DBUS_SESSION_BUS_ADDRESS
pids+=("$DBUS_SESSION_BUS_PID")
dbus-monitor --session > "$GGUIX_LOG/dbus.log" 2>&1 &
pids+=($!)

case "$mode" in
x11)
  n=$((90 + RANDOM % 100))
  Xvfb ":$n" -screen 0 1600x1000x24 -nolisten tcp > "$GGUIX_LOG/display.log" 2>&1 &
  pids+=($!)
  export DISPLAY=":$n"
  for _ in $(seq 50); do xdpyinfo >/dev/null 2>&1 && break; sleep 0.1; done
  openbox >> "$GGUIX_LOG/display.log" 2>&1 &
  pids+=($!)
  export GDK_BACKEND=x11 QT_QPA_PLATFORM=xcb
  ;;
wayland)
  export WAYLAND_DISPLAY=wl-gguix
  weston --backend=headless --renderer=pixman --socket=wl-gguix --width=1600 --height=1000 \
    > "$GGUIX_LOG/display.log" 2>&1 &
  pids+=($!)
  for _ in $(seq 50); do [ -S "$XDG_RUNTIME_DIR/wl-gguix" ] && break; sleep 0.1; done
  export GDK_BACKEND=wayland QT_QPA_PLATFORM=wayland
  ;;
sway)
  export WLR_BACKENDS=headless WLR_RENDERER=pixman WLR_LIBINPUT_NO_DEVICES=1
  export WAYLAND_DISPLAY=wayland-1
  cat > "$XDG_CONFIG_HOME/sway.conf" <<'EOC'
output HEADLESS-1 resolution 1600x1000
default_border none
for_window [app_id=".*"] floating enable
focus_follows_mouse no
EOC
  sway -c "$XDG_CONFIG_HOME/sway.conf" > "$GGUIX_LOG/display.log" 2>&1 &
  pids+=($!)
  for _ in $(seq 50); do [ -S "$XDG_RUNTIME_DIR/wayland-1" ] && break; sleep 0.1; done
  export SWAYSOCK=$(ls "$XDG_RUNTIME_DIR"/sway-ipc.*.sock 2>/dev/null | head -1)
  export GDK_BACKEND=wayland QT_QPA_PLATFORM=wayland
  # persistent virtual keyboard + pointer (see wlinput.py)
  python3 /repo/linux/wlinput.py serve > "$GGUIX_LOG/wlinput.log" 2>&1 &
  pids+=($!)
  sleep 1
  ;;
*) echo "unknown mode $mode" >&2; exit 2 ;;
esac

# Notification daemon: owns org.freedesktop.Notifications; dunst -print logs every shown notification.
dunst -print > "$GGUIX_LOG/dunst.log" 2>&1 &
pids+=($!)
sleep 0.5

# Write the session environment so another shell (docker exec) can attach: . $GGUIX_LOG/env.sh
{ for v in DISPLAY WAYLAND_DISPLAY SWAYSOCK GDK_BACKEND QT_QPA_PLATFORM DBUS_SESSION_BUS_ADDRESS XDG_RUNTIME_DIR XDG_CONFIG_HOME XDG_DATA_HOME XDG_CURRENT_DESKTOP PATH GGUIX_LOG GGUIX_MODE LIBGL_ALWAYS_SOFTWARE; do
    [ -n "${!v:-}" ] && echo "export $v='${!v}'"; done; } > "$GGUIX_LOG/env.sh"

if [ $# -eq 0 ]; then set -- bash; fi
"$@"
rc=$?
echo "session command exited rc=$rc" >> "$GGUIX_LOG/display.log"
exit $rc
