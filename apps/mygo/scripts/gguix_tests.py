#!/usr/bin/env python3
"""Aggregate Go, Vitest and Playwright results into the GGUIX_TESTS line (test category)."""
import json
import re
import sys

passed = failed = 0
nf = int(sys.argv[1]) if len(sys.argv) > 1 else 0

# Go: `go test -v` output
for line in open("bin/test-go.log", errors="replace"):
    if re.match(r"\s*--- PASS", line):
        passed += 1
    elif re.match(r"\s*--- FAIL", line):
        failed += 1

v = json.load(open("bin/test-vitest.json"))
passed += v["numPassedTests"]
failed += v["numFailedTests"]


def walk(suite):
    global passed, failed
    for spec in suite.get("specs", []):
        ok = spec["ok"] and all(r["status"] == "passed" for t in spec["tests"] for r in t["results"])
        if ok:
            passed += 1
        else:
            failed += 1
    for s in suite.get("suites", []):
        walk(s)


for s in json.load(open("frontend/e2e/results.json"))["suites"]:
    walk(s)
print(f"GGUIX_TESTS passed={passed} failed={failed} not_feasible={nf}")
sys.exit(1 if failed else 0)
