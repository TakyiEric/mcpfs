#!/bin/bash
set -euo pipefail

# env-audit.sh — Audit environment variables across Vercel projects
# Mounts: vercel
#
# Usage: ./examples/env-audit.sh /tmp/mnt/vercel

MNT="${1:?Usage: env-audit.sh <vercel-mount>}"
[ -f "$MNT/projects.json" ] || { echo "error: $MNT/projects.json not found" >&2; exit 1; }

echo "=== Environment Variable Audit ==="
echo

# Get project names from the JSON resource, then read each project's env
jq -r '.[].name // empty' "$MNT/projects.json" 2>/dev/null | while IFS= read -r project; do
  env_file="$MNT/projects/$project/env"
  [ -f "$env_file" ] || continue

  count=$(jq 'length' "$env_file" 2>/dev/null || echo 0)
  [ "$count" -gt 0 ] 2>/dev/null || continue

  echo "--- $project ($count vars) ---"
  jq -r '.[] | "  \(.key)  [\(.target | join(","))]"' "$env_file" 2>/dev/null | sort -u
  echo
done

echo "=== All Keys (deduplicated) ==="
jq -r '.[].name // empty' "$MNT/projects.json" 2>/dev/null | while IFS= read -r project; do
  env_file="$MNT/projects/$project/env"
  [ -f "$env_file" ] && jq -r '.[].key // empty' "$env_file" 2>/dev/null || true
done | sort -u | grep -v '^$'
