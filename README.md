# mcpfs

Mount any MCP server as a filesystem. Plan 9 for the agent era.

<!-- TODO: asciinema demo GIF -->

## The problem

The MCP ecosystem has 18,000+ servers. They inject 30,000–125,000 tokens of tool schemas before your agent asks a single question. mcpfs takes the opposite approach: MCP servers expose **resources** (not tools), and mcpfs mounts them as **files**. Agents read files. No schemas, no function calls, no token bloat.

```
cat /mnt/github/repos          # your repos, one per line
cat /mnt/vercel/deployments    # latest deploys
cat /mnt/docker/containers     # running containers
grep error /mnt/*/             # search across everything
```

## Quick start

```bash
go install github.com/airshelf/mcpfs/cmd/mcpfs@latest
go install github.com/airshelf/mcpfs/servers/github@latest

mkdir -p /tmp/mnt/github
mcpfs /tmp/mnt/github -- mcpfs-github
cat /tmp/mnt/github/repos
```

## Why files, not tools?

| | MCP Tools | MCP Resources | CLI | **Filesystem** |
|---|---|---|---|---|
| Discovery cost | 30K–125K tokens (schemas) | ~500 tokens (URI list) | ~200 tokens (--help) | **0 tokens** (`ls`) |
| How agents call it | function_call JSON | resources/read | subprocess + parse | **`cat` / `read`** |
| Cross-service query | N tool calls, N parsers | N reads, N parsers | N commands, N flags | **`grep -r pattern /mnt/`** |
| Composability | None | None | Pipes, but per-tool flags | **Full Unix: grep, awk, jq, diff** |
| Auth surface | Per-tool permissions | Per-server token | Per-CLI login | **Per-mount env var** |

## Available servers

8 servers, each 250–380 lines of Go.

| Server | Auth | Resources | Install |
|--------|------|-----------|---------|
| **mcpfs-github** | `GITHUB_TOKEN` | repos, issues, PRs, readme, actions, releases, notifications, gists | `go install .../servers/github@latest` |
| **mcpfs-vercel** | `VERCEL_TOKEN` | deployments, projects, env vars, domains, build/runtime logs | `go install .../servers/vercel@latest` |
| **mcpfs-docker** | Docker socket | containers, images, networks, volumes, logs, inspect | `go install .../servers/docker@latest` |
| **mcpfs-k8s** | `KUBECONFIG` | namespaces, pods, services, deployments, nodes, logs | `go install .../servers/k8s@latest` |
| **mcpfs-postgres** | `DATABASE_URL` | tables, schema, row counts, sample data, extensions, connections | `go install .../servers/postgres@latest` |
| **mcpfs-npm** | (none) | package info, versions, dependencies, maintainers, search | `go install .../servers/npm@latest` |
| **mcpfs-slack** | `SLACK_TOKEN` | channels, messages, threads, users, search | `go install .../servers/slack@latest` |
| **mcpfs-linear** | `LINEAR_API_KEY` | issues, projects, cycles, teams, labels, members | `go install .../servers/linear@latest` |

### Filesystem tree (GitHub example)

```
/mnt/github/
├── repos                          # all repos (name, stars, language)
├── notifications                  # unread notifications
├── gists                          # your gists
├── repos/owner/repo/issues        # issues for a repo
├── repos/owner/repo/pulls         # pull requests
├── repos/owner/repo/readme        # README content
├── repos/owner/repo/actions       # workflow runs
└── repos/owner/repo/releases      # releases
```

## Cross-service examples

**What's broken?** — Cross-reference GitHub issues with Vercel deploy errors:
```bash
# Mount both services
mcpfs /tmp/mnt/github -- mcpfs-github
mcpfs /tmp/mnt/vercel -- mcpfs-vercel

# Find failing deploys and related issues
grep ERROR /tmp/mnt/vercel/deployments
grep -i deploy /tmp/mnt/github/repos/myorg/myapp/issues
```

**Grep everything** — Search across all mounted services:
```bash
grep -r "database" /tmp/mnt/
```

**Project health dashboard** — Combine signals from multiple sources:
```bash
echo "=== Deploys ===" && cat /tmp/mnt/vercel/deployments | head -5
echo "=== Containers ===" && cat /tmp/mnt/docker/containers
echo "=== Open Issues ===" && cat /tmp/mnt/github/repos/myorg/myapp/issues | wc -l
```

See [examples/](examples/) for complete scripts.

## Benchmarks

| Metric | Filesystem | CLI | Raw MCP |
|--------|-----------|-----|---------|
| Discovery tokens | ~0 (ls) | ~200 (--help) | ~500 (resources/list) |
| Read tokens (repos) | ~500 | ~5000 | ~500 + framing |
| Composability | grep, awk, jq, diff | per-tool flags | custom JSON-RPC |
| Cross-service search | `grep -r` | N scripts | N clients |

See [bench/](bench/) for runnable benchmarks.

## Write your own server

Each server is a Go program using the `mcpserve` framework:

```go
package main

import "github.com/airshelf/mcpfs/pkg/mcpserve"

func main() {
    s := mcpserve.New("my-server", "0.1.0", func(uri string) (mcpserve.ReadResult, error) {
        switch uri {
        case "myservice://status":
            return mcpserve.ReadResult{Text: "all good"}, nil
        default:
            return mcpserve.ReadResult{}, fmt.Errorf("unknown: %s", uri)
        }
    })
    s.AddResource(mcpserve.Resource{
        URI:  "myservice://status",
        Name: "status",
    })
    s.Serve()
}
```

Mount it: `mcpfs /tmp/mnt/myservice -- my-server`

## How it works

```
 Agent / Shell                    mcpfs (FUSE)                MCP Server
┌─────────────┐              ┌──────────────────┐         ┌─────────────┐
│ cat repos   │─── read() ──▶│ FUSE → mcpclient │── RPC ─▶│ mcpserve    │
│ grep error  │              │ stdio JSON-RPC   │         │ resources/  │
│ ls /mnt/    │◀── bytes ───│ cache + format   │◀─ JSON ─│ read        │──▶ API
└─────────────┘              └──────────────────┘         └─────────────┘
```

1. `mcpfs` launches the MCP server as a child process (stdio transport)
2. On mount, it calls `resources/list` and `resources/templates/list` to build the directory tree
3. File reads trigger `resources/read` calls — responses become file content
4. Standard FUSE: works with any program that reads files

## Requirements

- Go 1.22+
- FUSE 3 (`libfuse3-dev` on Debian/Ubuntu, `macfuse` on macOS)
- Auth tokens for the services you want to mount (see table above)

## Project structure

```
cmd/mcpfs/          # FUSE mount CLI
pkg/mcpserve/       # MCP resource server framework (shared by all servers)
pkg/mcpclient/      # MCP client (JSON-RPC over stdio)
internal/fuse/      # FUSE filesystem implementation
servers/            # 8 MCP resource servers
  github/           # GitHub REST API
  vercel/           # Vercel REST API
  docker/           # Docker Engine API (unix socket)
  k8s/              # Kubernetes via kubectl
  postgres/         # PostgreSQL via database/sql
  npm/              # NPM registry API
  slack/            # Slack Web API
  linear/           # Linear GraphQL API
examples/           # Cross-service shell scripts
bench/              # Benchmarks (tokens, latency, composability)
```

## License

MIT
