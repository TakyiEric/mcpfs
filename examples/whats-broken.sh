#!/bin/bash
set -euo pipefail

# whats-broken.sh — Cross-reference GitHub issues with Vercel deploy errors
# Mounts: github, vercel
#
# Usage: ./examples/whats-broken.sh /tmp/mnt [owner/repo]

MNT="${1:?Usage: whats-broken.sh <mount-root> [owner/repo]}"
REPO="${2:-}"

for dir in "$MNT/github" "$MNT/vercel"; do
  [ -d "$dir" ] || { echo "error: $dir not mounted" >&2; exit 1; }
done

echo "=== Open GitHub Issues ==="
echo
if [ -n "$REPO" ]; then
  owner="${REPO%/*}"
  name="${REPO#*/}"
  issues="$MNT/github/repos/$owner/$name/issues"
  [ -f "$issues" ] && jq -r '.[]? | "  #\(.number) \(.title) [\(.state)]"' "$issues" 2>/dev/null || echo "  (no issues)"
else
  echo "  (specify owner/repo as second arg for issue details)"
fi
echo

echo "=== Failed Vercel Deployments ==="
echo
jq -r '.[] | select(.state != "READY") | "  \(.state)\t\(.url // .uid)\t\(.created | if . then (. / 1000 | todate) else "?" end)"' \
  "$MNT/vercel/deployments.json" 2>/dev/null || echo "  (none)"
echo

echo "=== Summary ==="
total=$(jq 'length' "$MNT/vercel/deployments.json" 2>/dev/null || echo 0)
failed=$(jq '[.[] | select(.state != "READY")] | length' "$MNT/vercel/deployments.json" 2>/dev/null || echo 0)
echo "Vercel: $failed/$total deploys not READY"
