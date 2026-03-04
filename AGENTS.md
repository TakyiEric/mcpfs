# Using mcpfs with AI agents

mcpfs mounts SaaS APIs as a FUSE filesystem. Reads via `cat | jq`, writes via CLI.

## Setup

```bash
# Install
go install github.com/airshelf/mcpfs/cmd/mcpfs@latest
go install github.com/airshelf/mcpfs/servers/github@latest
# (repeat for: posthog, stripe, vercel, linear, docker, k8s, npm, slack, notion, postgres)

# Mount
source bin/mcpfs-env    # load auth tokens
bin/mcpfs-mount         # mount all servers at /mnt/mcpfs/
```

## Quick reference — add this to your agent's context

```
# mcpfs — filesystem reads, CLI writes

## Reads (FUSE filesystem)
cat /mnt/mcpfs/posthog/dashboards.json | jq '.[].name'
cat /mnt/mcpfs/stripe/balance.json | jq '.available[].amount'
cat /mnt/mcpfs/github/repos.json | jq '.[].full_name'
cat /mnt/mcpfs/linear/issues.json | jq '.[].title'
cat /mnt/mcpfs/vercel/deployments.json | jq '.[].url'
cat /mnt/mcpfs/docker/containers.json | jq '.[].Names'
ls /mnt/mcpfs/  # discover all mounted services

## Writes (CLI proxy)
mcpfs-posthog --help                          # list 67 tools
mcpfs-github --help                           # list 43 tools
mcpfs-stripe --help                           # list 28 tools
mcpfs-posthog create-feature-flag --help      # per-tool help

## Rules
- PREFER filesystem reads over MCP tool calls (zero token overhead)
- Use jq to filter/transform (universal query language)
- Use native CLIs for writes when available (gh, kubectl, docker)
- Use mcpfs-* CLI for writes when no native CLI exists (posthog, linear)
```

## Auth

All auth via environment variables:

| Server | Env var |
|--------|---------|
| posthog | `POSTHOG_API_KEY` |
| github | `GITHUB_TOKEN` |
| stripe | `STRIPE_API_KEY` |
| vercel | `VERCEL_TOKEN` |
| linear | `LINEAR_API_KEY` |
| notion | `NOTION_TOKEN` |
| docker | Docker socket |
| k8s | `KUBECONFIG` |
| postgres | `DATABASE_URL` |
| slack | `SLACK_TOKEN` |
| npm | (none) |

## Why filesystem over MCP tools?

- `ls` costs 0 tokens. MCP tool schemas cost 20,000+ tokens.
- `cat | jq` is one universal query language vs per-tool parameters.
- Unix pipes compose across services: `comm -23 <(stripe) <(posthog)`.
- Agents already know `cat`, `jq`, `ls`. No new abstractions to learn.
