# mcpfs

Mount any MCP server as a filesystem. Reads via `cat`, writes via CLI.

## How it works

mcpfs connects to any MCP server (stdio or HTTP), classifies its tools into reads and writes, and exposes reads as files:

```
 Agent / Shell                    mcpfs (FUSE)                  Any MCP Server
┌─────────────┐              ┌──────────────────┐         ┌──────────────────┐
│ ls /mnt/...  │─── readdir ─▶│ tools/list →     │         │ PostHog, Stripe, │
│ cat file.json│─── read() ──▶│ classify → tree  │── RPC ─▶│ GitHub, Linear,  │
│ jq '.name'  │◀── bytes ───│ tools/call       │◀─ JSON ─│ or anything      │
└─────────────┘              └──────────────────┘         └──────────────────┘

 Agent / Shell                    mcpfs tool CLI
┌─────────────┐              ┌──────────────────┐
│ mcpfs tool   │── flags ──▶│ parse CLI flags  │── RPC ─▶ MCP Server
│ posthog      │              │ tools/call       │
│ create-flag  │◀── JSON ───│                  │◀─ JSON ─
└─────────────┘              └──────────────────┘
```

**Classification rules:**
- `list_*`, `get_all_*`, no required params → **file** (`dashboards.json`)
- `get_*`, `retrieve_*`, has required params → **directory** (lookup by ID)
- `create_*`, `update_*`, `delete_*` → **CLI only** (`mcpfs tool`)
- `search_*`, `query_*` → **CLI only** (`mcpfs tool`)

Resources (if the server has them) are also mounted as files.

## Quick start

```bash
# Build and install
go install github.com/airshelf/mcpfs/cmd/mcpfs@latest

# Auto-discover Claude Code plugins and mount in project dir
cd ~/src/myproject
mcpfs auto                    # mounts to .mcpfs/ in cwd
mcpfs auto --mount /mnt/mcpfs # or specify a custom mount dir

# Mount a single server
mcpfs .mcpfs/posthog --http https://mcp.posthog.com/mcp --auth "Bearer $POSTHOG_API_KEY"
mcpfs .mcpfs/stripe -- npx -y @stripe/mcp

# Read
ls .mcpfs/posthog/
cat .mcpfs/posthog/dashboards.json
cat .mcpfs/stripe/balance.json

# Write (CLI)
mcpfs tool posthog create-feature-flag --key my-flag --name "My Flag"
mcpfs tool stripe create_customer --name "Acme Corp" --email acme@example.com

# List all tools for a server
mcpfs tool posthog
mcpfs tool stripe

# Unmount
fusermount -u .mcpfs/posthog
```

## Config file

Mount multiple servers from a single config (`~/.config/mcpfs/servers.json`):

```json
{
  "posthog": {
    "type": "http",
    "url": "https://mcp.posthog.com/mcp",
    "headers": {"Authorization": "Bearer ${POSTHOG_API_KEY}"}
  },
  "stripe": {
    "command": "npx",
    "args": ["-y", "@stripe/mcp"],
    "env": {"STRIPE_SECRET_KEY": "${STRIPE_API_KEY}"}
  },
  "github": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"],
    "env": {"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"}
  }
}
```

Environment variables (`${VAR}`) are interpolated from the process environment or from `~/.config/mcpfs/env`.

```bash
# Mount all to .mcpfs/ in cwd
mcpfs --config ~/.config/mcpfs/servers.json

# Mount to custom dir
mcpfs --config ~/.config/mcpfs/servers.json --mount /mnt/mcpfs
```

## Auto-discover Claude Code plugins

If you use Claude Code, `mcpfs auto` discovers all installed MCP plugins and mounts them in your project:

```bash
cd ~/src/myproject
mcpfs auto           # discover + mount to .mcpfs/
mcpfs auto --json    # print discovered config (dry run)
```

Mounts to `.mcpfs/` in the current directory. Reads `.env.local` and `.env` from cwd for project-specific credentials (e.g., different Vercel teams, PostHog projects per repo).

It reads from all Claude Code config sources:
- `~/.claude.json` → `mcpServers` — global user-configured servers
- `~/.claude/plugins/` — installed plugins and their `.mcp.json`
- `~/.claude/settings.json` → `enabledPlugins` — also scans cache for these
- `~/.claude/.credentials.json` — OAuth tokens (Notion, etc.)
- `~/.config/mcpfs/servers.json` — additional user-defined servers
- `~/.config/mcpfs/env` — fallback env vars (API keys)
- `gh auth token` — GitHub token fallback

Non-data plugins (playwright, serena, context7) are skipped automatically.

## Cross-service composition

```bash
# Business dashboard
printf "%-20s %s\n" "Stripe balance" "$(cat .mcpfs/stripe/balance.json | jq -r '.available[] | "\(.currency) \(.amount / 100)"')"
printf "%-20s %s\n" "Active subs" "$(cat .mcpfs/stripe/subscriptions.json | jq '[.[] | select(.status=="active")] | length')"
printf "%-20s %s\n" "PH dashboards" "$(cat .mcpfs/posthog/dashboards.json | jq length)"

# Find paying customers with no analytics activity
comm -23 \
  <(cat .mcpfs/stripe/customers.json | jq -r '.[].email' | sort) \
  <(cat .mcpfs/posthog/events.json | jq -r '.[].distinct_id' | sort)
```

## Project structure

```
cmd/mcpfs/          # CLI: mount, tool, config, unmount
internal/
  config/           # servers.json parser with env interpolation
  fuse/             # FUSE filesystem (go-fuse/v2)
  toolfs/           # Tool classification and tree building
pkg/
  mcpclient/        # MCP client (stdio + HTTP transports)
  mcptool/          # Tool schema → CLI bridge
bin/
  mcpfs-mount       # Mount all servers from config
  mcpfs-migrate     # Migrate from Claude Code MCP plugins
```

## Requirements

- Go 1.22+
- FUSE 3 (`libfuse3-dev` on Debian/Ubuntu, `macfuse` on macOS)
- Auth tokens for the services you want to mount

## AI agent notes

- `ls` a mount to discover available data (0 tokens vs 20K+ for MCP tool schemas)
- `cat file.json | jq` for reads — standard JSON output
- `mcpfs tool <server>` to list write/query tools with `--help`
- `mcpfs tool <server> <tool> --flag value` for writes
- All tool output goes to stdout (JSON), hints go to stderr
- Exit codes: 0 success, 1 error

## License

MIT
