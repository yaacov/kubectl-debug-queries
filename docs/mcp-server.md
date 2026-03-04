# MCP Server

kubectl-debug-queries includes an MCP (Model Context Protocol) server that exposes Kubernetes resources, logs, and events to AI assistants.

## Modes

### Stdio (default)

For local AI assistant integration (Claude Desktop, Cursor IDE).

```bash
kubectl debug-queries mcp-server
```

### SSE (HTTP)

For network-accessible deployments (OpenShift Lightspeed, remote clients).

```bash
kubectl debug-queries mcp-server --sse --port 8080
kubectl debug-queries mcp-server --sse --port 8443 --cert-file tls.crt --key-file tls.key
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--sse` | `false` | Enable SSE mode over HTTP |
| `--port` | `9091` | Listen port |
| `--host` | `0.0.0.0` | Bind address |
| `--cert-file` | | TLS certificate (enables HTTPS) |
| `--key-file` | | TLS private key (enables HTTPS) |

## MCP Tools

The server exposes two tools:

### debug_read

Query Kubernetes resources, logs, and events. Subcommands:

| Command | Description | Key Flags |
|---------|-------------|-----------|
| `get` | Get a single resource | `resource`, `name`, `namespace`, `output`, `query` |
| `list` | List resources | `resource`, `namespace`, `all_namespaces`, `selector`, `sort_by`, `limit`, `output`, `query` |
| `logs` | Retrieve container logs | `name`, `namespace`, `container`, `previous`, `tail`, `since`, `sort_by`, `output`, `query` |
| `events` | List events | `namespace`, `all_namespaces`, `resource`, `name`, `sort_by`, `limit`, `output`, `query` |

The optional `query` flag accepts [TSL (Tree Search Language)](query-language.md) syntax for filtering, sorting, and field selection.

**Examples:**

```json
{"command": "get", "flags": {"resource": "pod", "name": "my-pod", "namespace": "default"}}
{"command": "list", "flags": {"resource": "pods", "namespace": "kube-system", "selector": "app=nginx"}}
{"command": "list", "flags": {"resource": "deployments", "all_namespaces": true, "limit": 20}}
{"command": "logs", "flags": {"name": "my-pod", "namespace": "default", "tail": 100, "sort_by": "time_desc"}}
{"command": "logs", "flags": {"name": "deployment/nginx", "namespace": "default", "tail": 100}}
{"command": "logs", "flags": {"name": "my-pod", "namespace": "default", "tail": 200, "output": "raw"}}
{"command": "logs", "flags": {"name": "my-pod", "namespace": "default", "tail": 200, "output": "json"}}
{"command": "events", "flags": {"namespace": "default", "resource": "Pod", "name": "my-pod"}}
```

**Query examples:**

```json
{"command": "list", "flags": {"resource": "pods", "namespace": "default", "query": "where Status = 'Running'"}}
{"command": "list", "flags": {"resource": "pods", "namespace": "default", "output": "json", "query": "select Name, Status where Restarts > 0"}}
{"command": "events", "flags": {"namespace": "default", "query": "where Type = 'Warning'"}}
{"command": "logs", "flags": {"name": "my-pod", "namespace": "default", "tail": 100, "query": "where level = 'ERROR'"}}
```

### debug_help

Get help for subcommands and their flags.

```json
{"command": "get"}
{"command": "list"}
{"command": "logs"}
{"command": "events"}
```

Omit command for an overview of all subcommands.

## AI Assistant Setup

### Claude Desktop

```bash
claude mcp add kubectl-debug-queries kubectl debug-queries mcp-server
```

### Cursor IDE

Settings → MCP → Add Server:
- **Name:** kubectl-debug-queries
- **Command:** kubectl
- **Args:** debug-queries mcp-server

### SSE Mode (remote)

Point the client at the SSE endpoint:

```
http://<host>:<port>/sse
```

## SSE Authentication

In SSE mode, per-session credentials are passed via HTTP headers:

| Header | Description |
|--------|-------------|
| `Authorization: Bearer <token>` | Bearer token for Kubernetes auth |
| `X-Kubernetes-Server: <url>` | Kubernetes API server URL |

**Precedence:** HTTP headers (per-session) > CLI flags > kubeconfig
