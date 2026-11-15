#!/bin/sh
# Runs go test (real IUP toolkit, window mapped but never shown) and prints the GGUIX_TESTS summary line.
cd "$(dirname "$0")" || exit 1
out=$(go test ${GOTAGS:+-tags $GOTAGS} -count=1 -v ./... 2>&1); status=$?
echo "$out" | grep -E '^(--- (FAIL|SKIP)|FAIL|panic|\s+app_test)' | head -40
passed=$(echo "$out" | grep -c -E '^--- PASS')
failed=$(echo "$out" | grep -c -E '^--- FAIL')
echo "GGUIX_TESTS passed=$passed failed=$failed not_feasible=0"
[ $status -eq 0 ]
