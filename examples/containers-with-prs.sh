#!/bin/bash
set -euo pipefail

# containers-with-prs.sh — Match Docker containers with GitHub PRs
# Mounts: docker, github
#
# Usage: ./examples/containers-with-prs.sh /tmp/mnt [owner/repo]

MNT="${1:?Usage: containers-with-prs.sh <mount-root> [owner/repo]}"
REPO="${2:-}"

for dir in "$MNT/docker" "$MNT/github"; do
  [ -d "$dir" ] || { echo "error: $dir not mounted" >&2; exit 1; }
done

echo "=== Running Docker Containers ==="
echo
jq -r '.[] | "  \(.Names[0] // .Id[:12])\t\(.Image)\t\(.State)"' \
  "$MNT/docker/containers.json" 2>/dev/null || echo "  (none)"
echo

if [ -n "$REPO" ]; then
  owner="${REPO%/*}"
  name="${REPO#*/}"
  pulls="$MNT/github/repos/$owner/$name/pulls"

  echo "=== Open PRs for $REPO ==="
  echo
  if [ -f "$pulls" ]; then
    jq -r '.[]? | "  #\(.number) \(.title) ← \(.head.ref // "?")"' "$pulls" 2>/dev/null || echo "  (none)"
  else
    echo "  (no pulls file)"
  fi
  echo

  echo "=== Cross-Reference ==="
  echo
  # Check if any container image/name contains a PR branch name
  if [ -f "$pulls" ]; then
    jq -r '.[]?.head.ref // empty' "$pulls" 2>/dev/null | while IFS= read -r branch; do
      match=$(jq -r ".[] | select((.Names[0] // \"\") + (.Image // \"\") | test(\"$branch\"; \"i\")) | .Names[0] // .Id[:12]" \
        "$MNT/docker/containers.json" 2>/dev/null || true)
      if [ -n "$match" ]; then
        echo "  Branch '$branch' → container $match"
      fi
    done
  fi
else
  echo "(specify owner/repo as second arg to cross-reference PRs)"
fi
