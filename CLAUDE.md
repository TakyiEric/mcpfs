# mcpfs

Mount MCP resource servers as FUSE filesystems.

## Quick start

```bash
go build ./...              # build everything
go vet ./...                # lint
go build -o mcpfs ./cmd/mcpfs  # build CLI
go build -o mcpfs-github ./servers/github  # build a server
```

## Architecture

- `pkg/mcpserve/` — MCP resource server framework. Every server uses it.
- `pkg/mcpclient/` — MCP JSON-RPC client over stdio.
- `internal/fuse/` — FUSE filesystem. Maps resources → dirs, templates → subdirs.
- `cmd/mcpfs/` — CLI entry point. Launches server, connects client, mounts FUSE.
- `servers/*/main.go` — Each server is 250–380 lines. Self-contained.

## Conventions

- Servers use `mcpserve.New()` → `AddResource()` / `AddTemplate()` → `Serve()`
- Auth from env vars (GITHUB_TOKEN, VERCEL_TOKEN, etc.)
- All output is pre-formatted text (not raw JSON) — optimized for cat/grep
- Read-only: no server writes to external APIs
- `slimObjects()` helper extracts key fields from JSON arrays (reuse across servers)
- SQL injection prevented via `pg_catalog` validation (postgres server)
- GQL injection prevented via escaping (linear server)

## Adding a new server

1. Create `servers/myservice/main.go`
2. Use `mcpserve.New("mcpfs-myservice", "0.1.0", readFunc)`
3. Add resources (static URIs) and templates (parameterized URIs)
4. `readFunc` switches on URI, calls API, returns formatted text
5. Build: `go build ./servers/myservice`
6. Test: `mcpfs /tmp/mnt -- ./mcpfs-myservice && cat /tmp/mnt/...`

## Testing

```bash
# Build all
go build ./...

# Mount and test (GitHub example)
mkdir -p /tmp/mnt/github
./mcpfs /tmp/mnt/github -- ./mcpfs-github
cat /tmp/mnt/github/repos
fusermount -u /tmp/mnt/github
```

## Dependencies

- `github.com/hanwen/go-fuse/v2` — FUSE bindings
- `github.com/lib/pq` — PostgreSQL driver (postgres server only)
- No other external dependencies. HTTP clients use `net/http`.
