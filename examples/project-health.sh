#!/bin/bash
set -euo pipefail

# project-health.sh — Combined health dashboard from multiple services
# Mounts: any combination of github, vercel, docker, k8s, postgres
#
# Usage: ./examples/project-health.sh /tmp/mnt

MNT="${1:?Usage: project-health.sh <mount-root>}"

echo "=== Project Health Dashboard ==="
echo "$(date -u '+%Y-%m-%d %H:%M UTC')"
echo

# GitHub
if [ -d "$MNT/github" ]; then
  echo "## GitHub"
  repos=$(jq 'length' "$MNT/github/repos.json" 2>/dev/null || echo "?")
  notifs=$(jq 'length' "$MNT/github/notifications.json" 2>/dev/null || echo "?")
  echo "  Repos:         $repos"
  echo "  Notifications: $notifs"
  echo
fi

# Vercel
if [ -d "$MNT/vercel" ]; then
  echo "## Vercel"
  total=$(jq 'length' "$MNT/vercel/deployments.json" 2>/dev/null || echo "?")
  ready=$(jq '[.[] | select(.state == "READY")] | length' "$MNT/vercel/deployments.json" 2>/dev/null || echo "?")
  failed=$(jq '[.[] | select(.state != "READY")] | length' "$MNT/vercel/deployments.json" 2>/dev/null || echo "?")
  projects=$(ls -d "$MNT/vercel/projects"/*/ 2>/dev/null | wc -l)
  domains=$(jq 'length' "$MNT/vercel/domains.json" 2>/dev/null || echo "?")
  echo "  Deployments:   $total ($ready ready, $failed other)"
  echo "  Projects:      $projects"
  echo "  Domains:       $domains"
  echo
fi

# Docker
if [ -d "$MNT/docker" ]; then
  echo "## Docker"
  containers=$(jq 'length' "$MNT/docker/containers.json" 2>/dev/null || echo "?")
  images=$(jq 'length' "$MNT/docker/images.json" 2>/dev/null || echo "?")
  volumes=$(jq 'length' "$MNT/docker/volumes.json" 2>/dev/null || echo "?")
  networks=$(jq 'length' "$MNT/docker/networks.json" 2>/dev/null || echo "?")
  echo "  Containers: $containers"
  echo "  Images:     $images"
  echo "  Volumes:    $volumes"
  echo "  Networks:   $networks"
  echo
fi

# Kubernetes
if [ -d "$MNT/k8s" ]; then
  echo "## Kubernetes"
  pods=$(jq 'length' "$MNT/k8s/pods.json" 2>/dev/null || echo "?")
  svcs=$(jq 'length' "$MNT/k8s/services.json" 2>/dev/null || echo "?")
  nodes=$(jq 'length' "$MNT/k8s/nodes.json" 2>/dev/null || echo "?")
  echo "  Pods:     $pods"
  echo "  Services: $svcs"
  echo "  Nodes:    $nodes"
  echo
fi

# PostgreSQL
if [ -d "$MNT/postgres" ]; then
  echo "## PostgreSQL"
  tables=$(jq 'length' "$MNT/postgres/tables.json" 2>/dev/null || echo "?")
  exts=$(jq 'length' "$MNT/postgres/extensions.json" 2>/dev/null || echo "?")
  echo "  Tables:     $tables"
  echo "  Extensions: $exts"
  echo
fi

echo "=== End Health Check ==="
