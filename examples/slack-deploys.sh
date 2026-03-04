#!/bin/bash
set -euo pipefail

# slack-deploys.sh — Find Slack messages about deployments + Vercel status
# Mounts: slack, vercel
#
# Usage: ./examples/slack-deploys.sh /tmp/mnt

MNT="${1:?Usage: slack-deploys.sh <mount-root>}"

for dir in "$MNT/slack" "$MNT/vercel"; do
  [ -d "$dir" ] || { echo "error: $dir not mounted" >&2; exit 1; }
done

echo "=== Recent Vercel Deployments ==="
echo
jq -r '.[:10][] | "\(.state)\t\(.url // .uid)\t\(.created | if . then (. / 1000 | todate) else "?" end)"' \
  "$MNT/vercel/deployments.json" 2>/dev/null || echo "  (none)"
echo

echo "=== Slack Deploy Mentions ==="
echo
for channel_dir in "$MNT/slack/channels"/*/; do
  [ -d "$channel_dir" ] || continue
  channel=$(basename "$channel_dir")
  messages="$channel_dir/messages"
  [ -f "$messages" ] || continue

  # Search message text for deploy keywords
  matches=$(jq -r '.[]? | select(.text // "" | test("deploy|shipped|released|rollback|revert"; "i")) | "\(.user // "?"): \(.text[:120])"' \
    "$messages" 2>/dev/null || true)
  if [ -n "$matches" ]; then
    echo "--- #$channel ---"
    echo "$matches" | head -10
    echo
  fi
done
