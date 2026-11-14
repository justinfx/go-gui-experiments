"""Browser frame probe for one app's web build (addenda, HARNESS_CHANGES E3).

Starts `make <target>` on the harness web port (as measure.py does), runs
bench/web.mjs, and writes results/<id>/web_frames.json. Does not touch metrics.json.
    python3 -I bench/web_frames.py <id> [target ...]   (default: bench-web)
"""
import json, os, signal, socket, subprocess, sys, time
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
PORT = int(os.environ.get("GGUIX_WEB_PORT", "8097"))

def listening():
    with socket.socket() as s:
        return s.connect_ex(("127.0.0.1", PORT)) == 0

app_id, targets = sys.argv[1], sys.argv[2:] or ["bench-web"]
res = ROOT / "results" / app_id
(res / "raw").mkdir(parents=True, exist_ok=True)
env = {k: v for k, v in os.environ.items() if not k.startswith("GGUIX_")}
env.update(GGUIX_DPR="2", PORT=str(PORT), GOTTH_PORT=str(PORT))
out = {}
for t in targets:
    if listening():
        sys.exit(f"port {PORT} busy")
    log = open(res / "raw" / f"web-frames-{t}-server.log", "w")
    p = subprocess.Popen(["make", t, f"PORT={PORT}", f"SERVE_PORT={PORT}", f"WEB_PORT={PORT}"], cwd=ROOT / "apps" / app_id,
                         env=env, stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
    t0 = time.time()
    while not listening():
        if time.time() - t0 > 900 or p.poll() is not None:
            out[t] = {"error": "server never came up"}
            break
        time.sleep(1)
    else:
        r = subprocess.run(["node", "web.mjs", f"http://localhost:{PORT}/?bench=1", "300000"], cwd=ROOT / "bench",
                           env=env, capture_output=True, text=True, timeout=420)
        d = json.loads(r.stdout[r.stdout.index("{"):])
        out[t] = {k: d.get(k) for k in ("browser_frames", "timed_out", "chrome_version", "dpr", "console_errors")}
    try:
        os.killpg(p.pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    time.sleep(2)
(res / "web_frames.json").write_text(json.dumps(out, indent=1) + "\n")
print(json.dumps(out, indent=1))
