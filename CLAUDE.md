# mcpfs

Mount any MCP server as a FUSE filesystem. Classifies tools into reads (files) and writes (CLI).

## Quick start

```bash
go build -o mcpfs ./cmd/mcpfs   # build
go test ./...                    # test
go vet ./...                     # lint
```

## Architecture

- `pkg/mcpclient/` — MCP JSON-RPC client (stdio + HTTP transports)
- `pkg/mcptool/` — Tool schema → CLI bridge. Parses JSON Schema into flags, dispatches calls.
- `internal/fuse/` — FUSE filesystem. Maps resources → files, templates → dirs.
- `internal/toolfs/` — Tool classification (list/get/create/update/delete/search/query)
- `internal/config/` — servers.json parser with env interpolation
- `cmd/mcpfs/` — CLI: mount, auto-discover, tool proxy, unmount

## How it works

mcpfs connects to any MCP server, classifies its tools, and exposes reads as files:

- `list_*`, `get_all_*`, no required params → **file** (`dashboards.json`)
- `get_*`, `retrieve_*`, has required params → **directory** (lookup by ID)
- `create_*`, `update_*`, `delete_*` → **CLI only** (`mcpfs tool`)
- `search_*`, `query_*` → **CLI only** (`mcpfs tool`)

Resources (if the server has them) are also mounted as files.

## Commands

```bash
mcpfs auto                              # discover Claude Code plugins, mount to .mcpfs/
mcpfs auto --json                       # dry run — show discovered config
mcpfs auto --mount /custom/dir          # mount to custom dir
mcpfs .mcpfs/posthog --http <url>       # mount single HTTP server
mcpfs .mcpfs/stripe -- npx -y @stripe/mcp  # mount single stdio server
mcpfs tool posthog                      # list tools for a server
mcpfs tool posthog create-flag --key x  # call a write tool
mcpfs -u .mcpfs/posthog                 # unmount
```

## Project-local mounts

`mcpfs auto` mounts to `.mcpfs/` in the current directory (project-local).
Reads `.env.local` and `.env` from cwd for project-specific credentials, then falls back to `~/.config/mcpfs/env`.

Different projects can have different API keys for the same services.

## Auto-discovery sources

1. `~/.config/mcpfs/servers.json` — user overrides (base)
2. `~/.claude.json` → `mcpServers` — global Claude Code servers
3. `~/.claude/plugins/installed_plugins.json` — installed plugins
4. `~/.claude/settings.json` → `enabledPlugins` — enabled plugins (cache scan)
5. `~/.claude/.credentials.json` — OAuth tokens
6. `~/.config/mcpfs/env` — fallback env vars
7. `gh auth token` — GitHub token fallback

Non-data plugins (playwright, serena, context7, skill plugins) are skipped.

## Key files

```
cmd/mcpfs/main.go       CLI routing, runConfig, runTool, loadEnvFile
cmd/mcpfs/auto.go       Plugin discovery, auth resolution, runAuto
internal/fuse/fs.go     FUSE inode tree (dirNode, fileNode, toolFileNode)
internal/toolfs/        Tool classification (segment-based verb matching)
pkg/mcpclient/          Stdio + HTTP MCP client
pkg/mcptool/            Tool schema → CLI flags → call
```

## Dependencies

- `github.com/hanwen/go-fuse/v2` — FUSE bindings
- No other external dependencies. HTTP clients use `net/http`.
