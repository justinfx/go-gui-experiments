#!/bin/sh
# Runs the T1..T12 headless tests (app.NewHeadless) and prints the summary.
cd "$(dirname "$0")" || exit 1
out=bin/test-output.txt
mkdir -p bin
CGO_ENABLED=0 go test -count=1 -v ./... 2>&1 | grep -v '^20[0-9][0-9]/[0-9][0-9]/' > "$out"
cat "$out"
pass=$(grep -c -- '--- PASS: TestT' "$out")
fail=$(grep -c -- '--- FAIL: TestT' "$out")
skip=$(grep -c -- '--- SKIP: TestT' "$out")
if grep -q -e '^FAIL' -e 'build failed' "$out" && [ "$fail" -eq 0 ]; then fail=1; fi
echo "GGUIX_TESTS passed=$pass failed=$fail not_feasible=$skip"
[ "$fail" -eq 0 ] && [ "$pass" -gt 0 ]
