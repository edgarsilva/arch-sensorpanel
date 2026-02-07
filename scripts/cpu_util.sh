#!/usr/bin/env bash
set -euo pipefail

read_cpu() {
  awk '/^cpu / {
    idle = $5
    total = $2 + $3 + $4 + $5 + $6 + $7 + $8
    print idle, total
  }' /proc/stat
}

read idle1 total1 < <(read_cpu)
sleep 0.5
read idle2 total2 < <(read_cpu)

awk -v i1="$idle1" -v t1="$total1" -v i2="$idle2" -v t2="$total2" '
BEGIN {
  idle = i2 - i1
  total = t2 - t1
  if (total <= 0) print 0
  else printf "%.1f\n", (100 * (total - idle) / total)
}'
