#!/usr/bin/env python3
"""Measure one GGUIX app. Run from the repo root, one app at a time:

    python3 -I bench/measure.py <id> [--runs 5] [--skip-cross] [--skip-web] [--only step,step]

Writes results/<id>/metrics.json and raw logs under results/<id>/raw/.
Steps: toolchain, plain_build, cold_build, incremental, sizes, idle_rss,
bench, tests, cgoprobe, cross, web, loc.
"""
WEBPORT = int(__import__("os").environ.get("GGUIX_WEB_PORT", "8097"))
import argparse
import datetime
import gzip
import hashlib
import json
import os
import plistlib
import re
import shutil
import signal
import socket
import statistics
import subprocess
import sys
import tempfile
import threading
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
WEBKIT = "com.apple.WebKit"

VENDOR = {  # library sources copied into apps by component CLIs (excluded from LOC, reported separately)
    "wails": "frontend/src/lib/components/ui/**",
    "mygo": "frontend/src/lib/components/ui/**",
    "gotth": "components/**,static/js/**",
}


def run(cmd, cwd, env=None, timeout=1800, log=None):
    t0 = time.time()
    try:
        p = subprocess.run(cmd, cwd=cwd, env=env, shell=isinstance(cmd, str), capture_output=True, text=True, timeout=timeout)
        rc, out = p.returncode, p.stdout + p.stderr
    except subprocess.TimeoutExpired as e:
        rc, out = -999, (e.stdout or b"").decode(errors="replace") if isinstance(e.stdout, bytes) else (e.stdout or "")
        out += "\nTIMEOUT"
    secs = round(time.time() - t0, 2)
    if log:
        log.write_text(f"$ {cmd}\n(rc={rc}, {secs}s)\n{out}")
    return rc, out, secs


def tail(s, n=12):
    return "\n".join(s.strip().splitlines()[-n:])


def ps_table():
    out = subprocess.run(["ps", "-A", "-o", "pid=,ppid=,rss=,comm="], capture_output=True, text=True).stdout
    rows = []
    for line in out.splitlines():
        parts = line.split(None, 3)
        if len(parts) == 4:
            rows.append((int(parts[0]), int(parts[1]), int(parts[2]), parts[3]))
    return rows


def tree_rss_kb(root_pid, webkit_baseline):
    rows = ps_table()
    children = {}
    for pid, ppid, rss, comm in rows:
        children.setdefault(ppid, []).append(pid)
    rss_of = {pid: rss for pid, _, rss, _ in rows}
    seen, stack = set(), [root_pid]
    while stack:
        p = stack.pop()
        if p in seen:
            continue
        seen.add(p)
        stack.extend(children.get(p, []))
    own = sum(rss_of.get(p, 0) for p in seen)
    webkit = sum(rss for pid, _, rss, comm in rows if WEBKIT in comm and pid not in webkit_baseline)
    return own, webkit


def webkit_pids():
    return {pid for pid, _, _, comm in ps_table() if WEBKIT in comm}


def metric_lines(text):
    out = []
    for line in text.splitlines():
        i = line.find("GGUIX_METRIC ")
        if i >= 0:
            try:
                out.append(json.loads(line[i + len("GGUIX_METRIC "):]))
            except json.JSONDecodeError:
                pass
    return out


def summarize(runs):
    """runs: list of {event: {field: value}} -> {event: {field: {median,min,max}}}"""
    events = {}
    for r in runs:
        for ev, fields in r.items():
            for k, v in fields.items():
                if isinstance(v, (int, float)) and not isinstance(v, bool):
                    events.setdefault(ev, {}).setdefault(k, []).append(v)
    return {ev: {k: {"median": statistics.median(v), "min": min(v), "max": max(v), "n": len(v)} for k, v in f.items()} for ev, f in events.items()}


def wait_port(port, timeout):
    t0 = time.time()
    while time.time() - t0 < timeout:
        with socket.socket() as s:
            s.settimeout(0.5)
            if s.connect_ex(("127.0.0.1", port)) == 0:
                return round(time.time() - t0, 2)
        time.sleep(0.5)
    return None


def kill_group(p):
    try:
        os.killpg(p.pid, signal.SIGTERM)
        p.wait(timeout=10)
    except Exception:
        try:
            os.killpg(p.pid, signal.SIGKILL)
        except Exception:
            pass


def set_marker(path, name, value):
    s = path.read_text()
    new = re.sub(r'(%s\s*=\s*")[^"]*(")' % name, r"\g<1>%s\g<2>" % value, s, count=1)
    if new == s and name == "UIMarker":
        new = re.sub(r"(UIMarker )\S+", r"\g<1>%s" % value, s, count=1)
    path.write_text(new)
    return new != s


def du_kb(path):
    if not path.exists():
        return None
    return int(subprocess.run(["du", "-sk", str(path)], capture_output=True, text=True).stdout.split()[0])


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


TREE_SKIP = {"bin", "node_modules", "dist", ".git", ".task"}


def tree_sha256(root, skip=TREE_SKIP):
    """sha256 over (relative path, file sha256) of every file under root, sorted, skipping dirs named in skip."""
    root = Path(root)
    h = hashlib.sha256()
    n = 0
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = sorted(d for d in dirnames if d not in skip)
        for fn in sorted(filenames):
            fp = Path(dirpath) / fn
            if fp.is_symlink() or not fp.is_file():
                continue
            h.update(str(fp.relative_to(root)).encode() + b"\0" + sha256_file(fp).encode() + b"\n")
            n += 1
    return {"sha256": h.hexdigest(), "files": n}


def artifact(path):
    """Provenance of one exercised artifact (audit C1): path, sha256, size, mtime. A directory (.app) gets a tree hash."""
    path = Path(path)
    if not path.exists():
        return {"path": str(path), "missing": True}
    st = path.stat()
    d = {"path": str(path.relative_to(ROOT)) if path.is_relative_to(ROOT) else str(path),
         "mtime": datetime.datetime.fromtimestamp(st.st_mtime, datetime.timezone.utc).isoformat()}
    if path.is_dir():
        t = tree_sha256(path, skip=set())
        d.update({"tree_sha256": t["sha256"], "files": t["files"], "size_kb": du_kb(path)})
    else:
        d.update({"sha256": sha256_file(path), "size": st.st_size})
    return d


def now_iso():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def listeners(port):
    """pids listening on TCP port (any address), via lsof."""
    out = subprocess.run(["lsof", "-nP", f"-iTCP:{port}", "-sTCP:LISTEN", "-t"], capture_output=True, text=True).stdout
    return sorted({int(x) for x in out.split()})


# Env vars the harness sets on purpose; every other inherited GGUIX_* var is scrubbed (audit C2).
GGUIX_KEEP = {"GGUIX_DPR"}


class Measure:
    def __init__(self, app_id, args):
        self.id, self.args = app_id, args
        self.app = ROOT / "apps" / app_id
        self.res = ROOT / "results" / app_id
        self.raw = self.res / "raw"
        self.raw.mkdir(parents=True, exist_ok=True)
        self.out_path = self.res / "metrics.json"
        self.m = json.loads(self.out_path.read_text()) if self.out_path.exists() else {}
        self.m["app"] = app_id
        self.base = app_id.split("-")[0]
        self.bin = self.app / "bin" / f"gguix-{self.base}"

    def env(self, **extra):
        e = {k: v for k, v in os.environ.items() if not k.startswith("GGUIX_") or k in GGUIX_KEEP}
        e.setdefault("GGUIX_DPR", "2")
        e["GGUIX_PREFS_DIR"] = tempfile.mkdtemp(prefix="gguix-prefs-")
        e.update({k: str(v) for k, v in extra.items()})
        return e

    def provenance(self):
        inherited = {k: v for k, v in os.environ.items() if k.startswith("GGUIX_")}
        passed = {k: v for k, v in self.env().items() if k.startswith("GGUIX_") and k != "GGUIX_PREFS_DIR"}
        self.m["provenance"] = {
            "recorded_at": now_iso(),
            "app_tree": tree_sha256(self.app),
            "core_tree": tree_sha256(ROOT / "core"),
            "gguix_env_inherited": inherited,
            "gguix_env_passed": passed,
            "gguix_env_scrubbed": sorted(k for k in inherited if k not in GGUIX_KEEP),
        }

    def record(self, step, **arts):
        self.m.setdefault("artifacts", {})[step] = {k: (artifact(v) if isinstance(v, Path) else v) for k, v in arts.items()}

    def save(self):
        self.out_path.write_text(json.dumps(self.m, indent=2, sort_keys=True))

    def make(self, target, timeout=1800, env=None, log=None):
        return run(["make", target], self.app, env or self.env(), timeout, log or self.raw / f"make-{target}.log")

    # --- steps ---
    def toolchain(self):
        t = {}
        for pj in [self.app / "package.json", self.app / "frontend" / "package.json"]:
            if pj.exists():
                d = json.loads(pj.read_text())
                nm = pj.parent / "node_modules"
                rc, out, _ = run("npm ls --all --parseable 2>/dev/null | wc -l", pj.parent)
                arc, aout, _ = run("npm audit --json", pj.parent, timeout=120)
                vulns = None
                try:
                    vulns = json.loads(aout).get("metadata", {}).get("vulnerabilities")
                except Exception:
                    pass
                t[str(pj.relative_to(self.app))] = {
                    "direct_deps": len(d.get("dependencies", {})),
                    "direct_dev_deps": len(d.get("devDependencies", {})),
                    "installed_packages": max(0, int(out.strip() or 0) - 1),
                    "node_modules_kb": du_kb(nm),
                    "npm_audit": vulns,
                }
        rc, out, secs = self.make("deps", timeout=3600)
        t["make_deps_warm"] = {"rc": rc, "secs": secs}
        self.m["toolchain"] = t

    def plain_build(self):
        self.make("clean")
        # Wails embeds frontend/dist and imports frontend bindings; GoTTH needs the templ
        # output. Generate those first, outside the timed part (audit A4).
        gen = None
        if self.base == "wails":
            gen = "wails3 task common:build:frontend"
        elif self.base == "mygo":
            gen = "frontend/node_modules/.bin/mygo generate && cd frontend && npm run build"
        elif self.base == "gotth" or self.id == "miqt-gotth":
            gen = "make gen"
        rec = {}
        if gen:
            grc, gout, gsecs = run(gen, self.app, self.env(), 1800, self.raw / "plain-go-build-gen.log")
            rec["gen"] = {"cmd": gen, "rc": grc, "secs": gsecs}
            if grc != 0:
                self.m["plain_go_build_from_clean"] = {**rec, "status": "n/a: needs generated sources", "tail": tail(gout, 6)}
                return
            rec["plain_build_note"] = f"generated sources were present: `{gen}` ran untimed after make clean, then go build ./... was timed"
        rc, out, secs = run("go build ./...", self.app, self.env(), 1800, self.raw / "plain-go-build.log")
        self.m["plain_go_build_from_clean"] = {**rec, "rc": rc, "secs": secs, "tail": tail(out, 6)}

    def cold_build(self):
        self.make("clean")
        cache = tempfile.mkdtemp(prefix="gguix-gocache-")
        try:
            rc, out, secs = self.make("build", timeout=3600, env=self.env(GOCACHE=cache), log=self.raw / "cold-build.log")
        finally:
            shutil.rmtree(cache, ignore_errors=True)
        self.m["cold_build"] = {"rc": rc, "secs": secs}
        rc, out, secs = self.make("build", log=self.raw / "warm-build.log")
        self.m["warm_noop_build"] = {"rc": rc, "secs": secs}
        self.record("cold_build", binary=self.bin)

    def incremental(self):
        inc = {}
        markers = [(self.app / "marker.go", "BuildMarker")]
        if self.base in ("wails", "mygo"):
            markers.append((self.app / "frontend/src/lib/marker.ts", "UIMarker"))
        if self.base == "gotth" or self.id == "miqt-gotth":
            markers.append((self.app / "views/marker.templ", "UIMarker"))
        for path, name in markers:
            if not path.exists():
                inc[name] = {"error": f"missing {path.relative_to(self.app)}"}
                continue
            orig = path.read_text()
            try:
                changed = set_marker(path, name, "1")
                rc, out, secs = self.make("build", log=self.raw / f"incremental-{name}.log")
                inc[name] = {"rc": rc, "secs": secs, "marker_changed": changed}
            finally:
                path.write_text(orig)
        self.make("build")
        self.m["incremental_build"] = inc
        self.record("incremental", binary=self.bin)

    def sizes(self):
        s = {}
        b = self.bin
        s["binary_bytes"] = b.stat().st_size if b.exists() else None
        bin_art = artifact(b)
        rc, out, secs = self.make("package", timeout=3600)
        s["package"] = {"rc": rc, "secs": secs}
        apps = sorted((self.app / "bin").glob("*.app"))
        if apps:
            appb = apps[0]
            s["app_bundle"] = str(appb.relative_to(self.app))
            s["app_bundle_kb"] = du_kb(appb)
            try:
                info = plistlib.loads((appb / "Contents/Info.plist").read_bytes())
                exe = appb / "Contents/MacOS" / info["CFBundleExecutable"]
                o = subprocess.run(["otool", "-L", str(exe)], capture_output=True, text=True).stdout
                libs = [l.strip().split(" (")[0] for l in o.splitlines()[1:]]
                s["external_dylibs"] = [l for l in libs if not (l.startswith("/System/") or l.startswith("/usr/lib/") or l.startswith("@"))]
                s["bundled_refs"] = [l for l in libs if l.startswith("@")]
                fw = appb / "Contents/Frameworks"
                s["bundled_frameworks_kb"] = du_kb(fw)
            except Exception as e:
                s["bundle_error"] = str(e)
        self.m["sizes"] = s
        self.record("sizes", binary=bin_art, **({"app_bundle": appb} if apps else {}))

    def idle_rss(self):
        b = self.bin
        self.record("idle_rss", binary=b)
        if not b.exists():
            self.m["idle_rss"] = {"error": "binary missing"}
            return
        base = webkit_pids()
        p = subprocess.Popen([str(b)], cwd=self.app, env=self.env(), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
        time.sleep(7)
        own, wk = tree_rss_kb(p.pid, base)
        kill_group(p)
        self.m["idle_rss"] = {"process_tree_mb": round(own / 1024, 1), "webkit_mb": round(wk / 1024, 1), "total_mb": round((own + wk) / 1024, 1)}

    def bench(self):
        # Build once, outside the sampling, then launch the built binary directly, the
        # same way idle_rss does, so peak memory is the app's own process tree only
        # (no make, wails3 task or .dev.app wrapper) and no stale binary is benched
        # (audit A1). For Wails this is bin/gguix-wails, the binary that `wails3 task
        # run` copies into bin/gguix-wails.dev.app and runs.
        rc, out, secs = self.make("build", log=self.raw / "bench-build.log")
        if rc != 0 or not self.bin.exists():
            self.m["bench"] = {"error": f"make build failed (rc={rc})", "tail": tail(out)}
            return
        self.record("bench", build={"rc": rc, "secs": secs}, binary=self.bin)
        runs, peaks, raw = [], [], []
        for i in range(self.args.runs):
            export = Path(tempfile.mkdtemp(prefix="gguix-export-")) / "export.csv"
            env = self.env(GGUIX_BENCH="1", GGUIX_BENCH_EXPORT=export)
            base = webkit_pids()
            p = subprocess.Popen([str(self.bin)], cwd=self.app, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, start_new_session=True)
            peak_own = peak_wk = 0
            t0 = time.time()
            lines = []
            th = threading.Thread(target=lambda: lines.extend(p.stdout), daemon=True)
            th.start()
            while p.poll() is None and time.time() - t0 < self.args.bench_timeout:
                own, wk = tree_rss_kb(p.pid, base)
                peak_own, peak_wk = max(peak_own, own), max(peak_wk, wk)
                if any('"event":"done"' in l for l in lines):
                    time.sleep(2)
                    break
                time.sleep(0.1)
            kill_group(p)
            th.join(timeout=5)
            text = "".join(lines)
            (self.raw / f"bench-run{i + 1}.log").write_text(text)
            ev = {m["event"]: {k: v for k, v in m.items() if k != "event"} for m in metric_lines(text)}
            runs.append(ev)
            peaks.append({"peak_process_tree_mb": round(peak_own / 1024, 1), "peak_webkit_mb": round(peak_wk / 1024, 1), "peak_total_mb": round((peak_own + peak_wk) / 1024, 1)})
            raw.append(sorted(ev.keys()))
        self.m["bench"] = {
            "runs": self.args.runs,
            "launch": f"bin/{self.bin.name} (direct, after one make build)",
            "events_seen": raw,
            "summary": summarize(runs),
            "memory": {k: {"median": statistics.median([p[k] for p in peaks]), "max": max(p[k] for p in peaks)} for k in peaks[0]},
            "errors": [r.get("error") for r in runs if "error" in r],
        }

    def tests(self):
        rc, out, secs = self.make("test", timeout=2400)
        m = re.findall(r"GGUIX_TESTS passed=(\d+) failed=(\d+) not_feasible=(\d+)", out)
        self.m["tests"] = {"rc": rc, "secs": secs, "line": dict(zip(["passed", "failed", "not_feasible"], map(int, m[-1]))) if m else None}

    def cgoprobe(self):
        rc, out, secs = self.make("cgoprobe", timeout=3600)
        res = [{"target": t, "result": r, "reason": why.strip()} for t, r, why in re.findall(r"CGOPROBE (\S+) (PASS|FAIL)(.*)", out)]
        self.m["cgoprobe"] = {"rc": rc, "secs": secs, "results": res}
        self.make("build")

    def cross(self):
        c = {}
        for t in ["cross-windows", "cross-linux"]:
            rc, out, secs = self.make(t, timeout=3600)
            c[t] = {"rc": rc, "secs": secs, "status": "UNSUPPORTED" if rc == 3 or "UNSUPPORTED:" in out else ("PASS" if rc == 0 else "FAIL"), "tail": tail(out, 8)}
        self.m["cross"] = c

    def web_served(self):
        """The served dir and the hashes of index.html and the wasm/js bundle (audit C1)."""
        d = {}
        for cand in [self.app / "bin" / "web", self.app / "frontend" / "dist", self.app / "static"]:
            if cand.exists():
                d["served_dir"] = str(cand.relative_to(ROOT))
                idx = cand / "index.html"
                if idx.exists():
                    d["index_html_sha256"] = sha256_file(idx)
                d["bundle_sha256"] = {str(f.relative_to(cand)): sha256_file(f) for f in sorted(list(cand.rglob("*.wasm")) + list(cand.rglob("*.js")))}
                break
        if self.base == "wails":
            d["server_binary"] = artifact(self.app / "bin" / "gguix-wails-server")
        elif self.base == "gotth" or self.id == "miqt-gotth":
            d["server_binary"] = artifact(self.app / "bin" / "gguix-gotth-web")
        return d

    def web_bundle(self):
        webdir = self.app / "bin" / "web"
        if not webdir.exists():
            return {}
        wasm = list(webdir.rglob("*.wasm"))
        return {"bundle_dir_kb": du_kb(webdir),
                "wasm_files": {str(f.relative_to(webdir)): {"raw_kb": f.stat().st_size // 1024, "gzip_kb": len(gzip.compress(f.read_bytes(), 6)) // 1024} for f in wasm}}

    def web(self):
        w = {}
        targets = ["bench-web"] + (["web-remote"] if self.base in ("fyne", "cogent", "mygoui") else [])
        for t in targets:
            busy = listeners(WEBPORT)
            if busy:
                w[t] = {"error": f"port {WEBPORT} already has a listener before start (pids {busy})"}
                continue
            started = now_iso()
            p = subprocess.Popen(["make", t, f"PORT={WEBPORT}", f"SERVE_PORT={WEBPORT}", f"WEB_PORT={WEBPORT}"], cwd=self.app, env={**self.env(), "PORT": str(WEBPORT), "GOTTH_PORT": str(WEBPORT)}, stdout=open(self.raw / f"make-{t}.log", "w"), stderr=subprocess.STDOUT, start_new_session=True)
            up = wait_port(WEBPORT, 900)
            if up is None:
                kill_group(p)
                code = p.returncode
                w[t] = {"error": f"port {WEBPORT} never opened", "rc": code}
                continue
            owners = {}
            for pid in listeners(WEBPORT):
                try:
                    owners[pid] = os.getpgid(pid)
                except OSError:
                    owners[pid] = None
            if not owners or any(g != p.pid for g in owners.values()):
                kill_group(p)
                w[t] = {"error": f"listener on {WEBPORT} is not in the server's process group {p.pid}: {owners}", "server_ready_secs": up}
                continue
            if up < 0.3:
                kill_group(p)
                w[t] = {"error": f"server ready in {up}s (< 0.3 s), suggests a stale listener", "server_ready_secs": up}
                continue
            time.sleep(2)
            # Recorded right after this target's build, before the next target rebuilds (audit A6).
            bundle = self.web_bundle()
            served = self.web_served()
            rc, out, secs = run(["node", str(ROOT / "bench/web.mjs"), f"http://localhost:{WEBPORT}/?bench=1", "300000"], ROOT / "bench", env=self.env(), timeout=420, log=self.raw / f"web-{t}.log")
            kill_group(p)
            time.sleep(2)
            try:
                data = json.loads(out[out.index("{"):])
            except Exception:
                data = {"error": tail(out)}
            data["server_ready_secs"] = up
            data["events"] = {m["event"]: {k: v for k, v in m.items() if k != "event"} for m in data.pop("metrics", [])}
            data.update(bundle)
            data["provenance"] = {**served, "started": started, "ended": now_iso()}
            w[t] = data
        self.m["web"] = w

    def loc(self):
        args = ["go", "run", "./loc", "-app", str(self.app)]
        cfg = self.res / "loc_args.json"
        vendor, custom = VENDOR.get(self.id, ""), ""
        if cfg.exists():
            c = json.loads(cfg.read_text())
            vendor, custom = c.get("vendor", vendor), c.get("custom", "")
        args += ["-vendor", vendor, "-custom", custom]
        rc, out, _ = run(args, ROOT / "tools", timeout=300)
        self.m["loc"] = json.loads(out) if rc == 0 else {"error": tail(out)}


STEPS = ["toolchain", "plain_build", "cold_build", "incremental", "sizes", "idle_rss", "bench", "tests", "cgoprobe", "cross", "web", "loc"]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("id")
    ap.add_argument("--runs", type=int, default=5)
    ap.add_argument("--bench-timeout", type=int, default=300)
    ap.add_argument("--skip-cross", action="store_true")
    ap.add_argument("--skip-web", action="store_true")
    ap.add_argument("--only", default="")
    a = ap.parse_args()
    m = Measure(a.id, a)
    steps = [s for s in (a.only.split(",") if a.only else STEPS) if s]
    if a.skip_cross:
        steps = [s for s in steps if s != "cross"]
    # miqt-gotth has a web target (the GoTTH server); the other MIQT apps do not (audit B8).
    if a.skip_web or (a.id.split("-")[0] == "miqt" and a.id != "miqt-gotth") or a.id.split("-")[0] == "mygo":
        steps = [s for s in steps if s != "web"]
    m.provenance()
    for s in steps:
        t0 = time.time()
        m.m.setdefault("step_times", {})[s] = {"start": now_iso()}
        print(f"[{a.id}] {s} ...", flush=True)
        try:
            getattr(m, s)()
        except Exception as e:
            m.m.setdefault("step_errors", {})[s] = repr(e)
        m.m.setdefault("step_secs", {})[s] = round(time.time() - t0, 1)
        m.m["step_times"][s]["end"] = now_iso()
        m.save()
    print(f"[{a.id}] wrote {m.out_path.relative_to(ROOT)}")


if __name__ == "__main__":
    sys.exit(main())
