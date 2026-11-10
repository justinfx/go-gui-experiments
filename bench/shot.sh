#!/bin/sh
# Capture ONE app window only. Never the full desktop.
# Usage: bench/shot.sh <process-name-substring> <out.png>
# macOS: finds the largest on-screen window owned by a process whose name
# contains the substring, then runs screencapture -l <windowid>.
# Linux (Xvfb in Docker): uses xdotool + import on that window only.
# Exits non-zero, capturing nothing, if no matching window is found.
set -eu
name="$1"; out="$2"
case "$(uname)" in
Darwin)
  wid=$(swift -e '
import CoreGraphics
let want = CommandLine.arguments[1].lowercased()
let list = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as! [[String: Any]]
var best: (Int, Double)? = nil
for w in list {
  guard let owner = (w["kCGWindowOwnerName"] as? String)?.lowercased(), owner.contains(want),
        (w["kCGWindowLayer"] as? Int) == 0,
        let id = w["kCGWindowNumber"] as? Int,
        let b = w["kCGWindowBounds"] as? [String: Double] else { continue }
  let area = (b["Width"] ?? 0) * (b["Height"] ?? 0)
  if best == nil || area > best!.1 { best = (id, area) }
}
if let b = best { print(b.0) }
' "$name")
  [ -n "$wid" ] || { echo "shot.sh: no window for '$name'; nothing captured" >&2; exit 2; }
  screencapture -x -o -l "$wid" "$out"
  ;;
Linux)
  wid=$(xdotool search --onlyvisible --class "$name" | tail -1 || true)
  [ -n "$wid" ] || { echo "shot.sh: no window for '$name'; nothing captured" >&2; exit 2; }
  import -window "$wid" "$out"
  ;;
esac
echo "$out"
