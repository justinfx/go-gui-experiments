#!/bin/sh
# Runs go test and prints the GGUIX_TESTS summary line.
cd "$(dirname "$0")" || exit 1
out=$(go test -count=1 -v ./... 2>&1); status=$?
echo "$out" | grep -E '^(--- (FAIL|SKIP)|FAIL|panic|\s+app_test)' | head -40
passed=$(echo "$out" | grep -c -E '^--- PASS')
failed=$(echo "$out" | grep -c -E '^--- FAIL')
nf=$(cat not_feasible.count 2>/dev/null || echo 0)
echo "GGUIX_TESTS passed=$passed failed=$failed not_feasible=$nf"
[ $status -eq 0 ]
