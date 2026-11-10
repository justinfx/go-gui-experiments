#!/usr/bin/env python3
"""wlinput: persistent virtual keyboard + pointer for headless wlroots compositors (sway).

Why: wtype builds a throwaway keymap with non-evdev keycodes, which GLFW (Fyne) and Qt map
wrongly, and no packaged tool provides a virtual pointer. This speaks the Wayland wire protocol
directly (zwp_virtual_keyboard_v1, zwlr_virtual_pointer_v1) and keeps the devices alive.

  wlinput.py serve                      start the daemon (needs WAYLAND_DISPLAY, XDG_RUNTIME_DIR)
  wlinput.py key ctrl+shift+c           press a chord (client; talks to the daemon over a FIFO)
  wlinput.py type "hello"               type ASCII text
  wlinput.py click X Y [left|right]     absolute click in compositor layout coordinates
  wlinput.py move X Y
"""
import os, socket, struct, subprocess, sys, tempfile, time, array

EXTENT = (1600, 1000)
FIFO = os.path.join(os.environ.get("XDG_RUNTIME_DIR", "/tmp"), "wlinput.fifo")

EV = {n: c for c, n in {
    1: "esc", 2: "1", 3: "2", 4: "3", 5: "4", 6: "5", 7: "6", 8: "7", 9: "8", 10: "9", 11: "0", 12: "minus", 13: "equal",
    14: "backspace", 15: "tab", 16: "q", 17: "w", 18: "e", 19: "r", 20: "t", 21: "y", 22: "u", 23: "i", 24: "o", 25: "p",
    28: "return", 30: "a", 31: "s", 32: "d", 33: "f", 34: "g", 35: "h", 36: "j", 37: "k", 38: "l", 44: "z", 45: "x",
    46: "c", 47: "v", 48: "b", 49: "n", 50: "m", 51: "comma", 52: "period", 53: "slash", 57: "space", 59: "f1", 60: "f2",
    61: "f3", 62: "f4", 63: "f5", 64: "f6", 65: "f7", 66: "f8", 67: "f9", 68: "f10", 87: "f11", 88: "f12",
    102: "home", 103: "up", 104: "pageup", 105: "left", 106: "right", 107: "end", 108: "down", 109: "pagedown", 111: "delete",
}.items()}
MODS = {"shift": (42, 1), "ctrl": (29, 4), "alt": (56, 8), "super": (125, 64)}  # evdev code, xkb mask (us layout)
SHIFTED = {c: n for c, n in zip('!@#$%^&*()', '1234567890')}


class Wl:
    def __init__(self):
        path = os.path.join(os.environ["XDG_RUNTIME_DIR"], os.environ["WAYLAND_DISPLAY"])
        self.s = socket.socket(socket.AF_UNIX); self.s.connect(path)
        self.next = 2; self.buf = b""
        self.reg = self.new(); self.send(1, 1, struct.pack("I", self.reg))
        self.globals = {}
        self.roundtrip()

    def new(self):
        i = self.next; self.next += 1; return i

    def send(self, obj, op, body=b"", fds=()):
        msg = struct.pack("II", obj, ((8 + len(body)) << 16) | op) + body
        if fds:
            self.s.sendmsg([msg], [(socket.SOL_SOCKET, socket.SCM_RIGHTS, array.array("i", fds))])
        else:
            self.s.sendall(msg)

    def events(self, block_until_done=None):
        out = []
        while True:
            while len(self.buf) >= 8:
                obj, w = struct.unpack("II", self.buf[:8]); size = w >> 16
                if len(self.buf) < size: break
                out.append((obj, w & 0xffff, self.buf[8:size])); self.buf = self.buf[size:]
            if block_until_done is not None and any(o == block_until_done for o, _, _ in out): return out
            if block_until_done is None and not self.buf: return out
            d = self.s.recv(65536)
            if not d: return out
            self.buf += d

    def roundtrip(self):
        cb = self.new(); self.send(1, 0, struct.pack("I", cb))
        for obj, op, body in self.events(block_until_done=cb):
            if obj == self.reg and op == 0:
                name, ln = struct.unpack("II", body[:8]); iface = body[8:8 + ln - 1].decode()
                ver = struct.unpack("I", body[8 + ((ln + 3) & ~3):][:4])[0]
                self.globals[iface] = (name, ver)

    def bind(self, iface, ver=1):
        name, v = self.globals[iface]; ver = min(ver, v)
        i = self.new(); b = iface.encode() + b"\0"; b += b"\0" * (-len(b) % 4)
        self.send(self.reg, 0, struct.pack("II", name, len(iface) + 1) + b + struct.pack("II", ver, i))
        return i


def serve():
    w = Wl()
    seat = w.bind("wl_seat", 1)
    vkm = w.bind("zwp_virtual_keyboard_manager_v1")
    vpm = w.bind("zwlr_virtual_pointer_manager_v1")
    kb = w.new(); w.send(vkm, 0, struct.pack("II", seat, kb))
    ptr = w.new(); w.send(vpm, 0, struct.pack("II", seat, ptr))
    km = subprocess.check_output(["xkbcli", "compile-keymap", "--layout", "us"]) + b"\0"
    f = tempfile.TemporaryFile(); f.write(km); f.flush()
    w.send(kb, 0, struct.pack("II", 1, len(km)), fds=[f.fileno()])
    w.roundtrip()
    t0 = time.time()
    ms = lambda: int((time.time() - t0) * 1000) & 0xffffffff
    mask = [0]

    def setmods(m):
        mask[0] = m; w.send(kb, 2, struct.pack("IIII", m, 0, 0, 0))

    def key(code, down):
        w.send(kb, 1, struct.pack("III", ms(), code, 1 if down else 0))

    def chord(spec):
        parts = spec.split("+"); name = parts[-1].lower(); mods = [m.lower() for m in parts[:-1]]
        for m in mods:
            key(MODS[m][0], True); setmods(mask[0] | MODS[m][1])
        key(EV[name], True); time.sleep(0.03); key(EV[name], False)
        for m in reversed(mods):
            key(MODS[m][0], False); setmods(mask[0] & ~MODS[m][1])

    def typ(text):
        for ch in text:
            if ch == " ": chord("space")
            elif ch == "\n": chord("return")
            elif ch.isupper(): chord("shift+" + ch.lower())
            elif ch == "/": chord("slash")
            elif ch == ".": chord("period")
            elif ch == "-": chord("minus")
            elif ch == ",": chord("comma")
            elif ch == "_": chord("shift+minus")
            else: chord(ch)
            time.sleep(0.01)

    def move(x, y):
        w.send(ptr, 1, struct.pack("IIIII", ms(), int(x), int(y), *EXTENT)); w.send(ptr, 4)

    def click(x, y, btn):
        move(x, y); time.sleep(0.05)
        code = {"left": 0x110, "right": 0x111, "middle": 0x112}[btn]
        w.send(ptr, 2, struct.pack("III", ms(), code, 1)); w.send(ptr, 4); time.sleep(0.05)
        w.send(ptr, 2, struct.pack("III", ms(), code, 0)); w.send(ptr, 4)

    if os.path.exists(FIFO): os.unlink(FIFO)
    os.mkfifo(FIFO)
    print("wlinput ready", flush=True)
    fd = os.open(FIFO, os.O_RDWR)
    with os.fdopen(fd) as fifo:
        for line in fifo:
            a = line.rstrip("\n").split(" ", 1)
            cmd, arg = a[0], (a[1] if len(a) > 1 else "")
            try:
                if cmd == "key": chord(arg)
                elif cmd == "type": typ(arg)
                elif cmd == "move": x, y = arg.split(); move(x, y)
                elif cmd == "click":
                    p = arg.split(); click(p[0], p[1], p[2] if len(p) > 2 else "left")
                elif cmd == "quit": return
                w.roundtrip()
            except Exception as e:
                print("wlinput error:", cmd, arg, e, flush=True)


def client(argv):
    with open(FIFO, "w") as f:
        f.write(" ".join(argv) + "\n")
    time.sleep(0.25)


if __name__ == "__main__":
    if len(sys.argv) < 2: sys.exit(__doc__)
    if sys.argv[1] == "serve": serve()
    else: client(sys.argv[1:])
