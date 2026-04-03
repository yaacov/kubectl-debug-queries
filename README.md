# kubectl-debug-queries

Query Kubernetes resources, logs, and events using SQL like query language — as a CLI and an MCP server for AI assistants.

## Installation

Install the latest release (Linux / macOS):

```bash
curl -sSL https://raw.githubusercontent.com/yaacov/kubectl-debug-queries/main/install.sh | bash
```

This downloads the binary, verifies its checksum, and sets up shell completion. Installs to `~/.local/bin` by default.

Or install with [krew](https://krew.sigs.k8s.io/):

```bash
kubectl krew install debug-queries
```

Or build from source:

```bash
make build
cp kubectl-debug-queries ~/.local/bin/kubectl-debug_queries
```

Once installed, kubectl discovers it as a plugin:

```bash
kubectl debug-queries --help
```

For more options (manual download, shell completion, uninstall), see [Installation](docs/installation.md).

## Quick Start

Query resources, logs and events using an SQL like [query language](docs/query-language.md).

```bash
# Get pod logs (newest first, auto-detected smart format)
kubectl debug-queries logs --name my-pod --namespace default --tail 100 --sort-by time_desc

# Get deployment logs (automatically selects a running pod)
kubectl debug-queries logs --name deployment/nginx --namespace default --tail 100

# Get raw unprocessed logs
kubectl debug-queries logs --name my-pod --namespace default --tail 100 --output raw

# List events for a specific resource
kubectl debug-queries events --namespace default --resource Pod --name my-pod

# Filter with TSL query language ("where" keyword is optional)
kubectl debug-queries list --resource pods --namespace default --query "Status = 'Running'"

# JSON output with field selection
kubectl debug-queries list --resource pods --namespace default --output json \
  --query "select Name, Status where Restarts > 0"

# Filter logs by level
kubectl debug-queries logs --name deployment/nginx --namespace default --tail 100 \
  --query "level = 'ERROR'"

# Filter by nested log field
kubectl debug-queries logs --name deployment/forklift-controller \
  --namespace konveyor-forklift --tail 100 --query "fields.map is not null"

# MCP server (stdio, for Claude Desktop / Cursor IDE)
kubectl debug-queries mcp-server

# MCP server (SSE, for OpenShift Lightspeed)
kubectl debug-queries mcp-server --sse --port 8080

# MCP server from container image
podman run --rm -p 8080:8080 \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  quay.io/yaacov/kubectl-debug-queries-mcp-server:latest
```

## AI Assistant Setup

**Claude Desktop:**

```bash
claude mcp add kubectl-debug-queries kubectl debug-queries mcp-server
```

**Cursor IDE:**

Settings → MCP → Add Server → Name: `kubectl-debug-queries`, Command: `kubectl`, Args: `debug-queries mcp-server`

## Authentication

Uses standard kubectl flags (`--kubeconfig`, `--context`, `--token`, `--server`). Supports all kubeconfig authentication methods including bearer tokens, client certificates, exec providers, and OIDC.

## Commands

All commands use named flags only — no positional arguments. The CLI and MCP tools share the exact same API.

| Command | Description | Required Flags |
|---------|-------------|----------------|
| `get` | Get a single resource | `--resource`, `--name`, `--namespace` * |
| `list` | List resources | `--resource`, `--namespace` (or `--all-namespaces` / `-A`) * |
| `logs` | Retrieve container logs | `--name`, `--namespace` |
| `events` | List events | `--namespace` (or `--all-namespaces` / `-A`) |

\* `get` and `list` also accept positional arguments as an alternative to `--resource`/`--name` flags (see [CLI Usage](docs/cli-usage.md)).

## Deploy on OpenShift

```bash
make deploy              # Deployment + Service
make deploy-route        # External Route with TLS
make deploy-olsconfig    # Register with OpenShift Lightspeed
```

## Documentation

See the [docs/](docs/) directory for detailed guides:

- [Installation](docs/installation.md)
- [CLI Usage](docs/cli-usage.md)
- [Query Language (TSL)](docs/query-language.md)
- [MCP Server](docs/mcp-server.md)
- [Containerized](docs/containerized.md)
- [Authentication](docs/authentication.md)
- [Deployment](docs/deployment.md)

## Development

```bash
make help          # Show all targets
make build         # Build binary
make test          # Run tests
make fmt           # Format code
make vet           # Run go vet
make vendor        # Populate vendor/
make image-build-amd64   # Build container image
```

## License

Apache License 2.0
