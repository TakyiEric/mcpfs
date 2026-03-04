#!/bin/bash
set -euo pipefail

# grep-everything.sh — Search across all mounted MCP services at once
# Mounts: any combination
#
# Usage: ./examples/grep-everything.sh /tmp/mnt <pattern>

MNT="${1:?Usage: grep-everything.sh <mount-root> <pattern>}"
PATTERN="${2:?Usage: grep-everything.sh <mount-root> <pattern>}"

echo "=== Searching all mounts for: $PATTERN ==="
echo

found=0
for service_dir in "$MNT"/*/; do
  [ -d "$service_dir" ] || continue
  service=$(basename "$service_dir")

  # Recursively grep all readable files under this mount
  matches=$(grep -ri "$PATTERN" "$service_dir" 2>/dev/null || true)
  if [ -n "$matches" ]; then
    echo "--- $service ---"
    echo "$matches" | head -20
    echo
    found=$((found + $(echo "$matches" | wc -l)))
  fi
done

if [ "$found" -eq 0 ]; then
  echo "No matches found for '$PATTERN' across $(ls -d "$MNT"/*/ 2>/dev/null | wc -l) mounts."
else
  echo "=== $found matches across all services ==="
fi
