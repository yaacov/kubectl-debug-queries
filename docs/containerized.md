# Running the MCP Server from a Container Image

Run the kubectl-debug-queries MCP server as a container — locally with Podman/Docker, or on any container runtime — without installing Go or building from source.

## Image

The pre-built image is available at:

```
quay.io/yaacov/kubectl-debug-queries-mcp-server:latest
```

The image is multi-arch (`linux/amd64` and `linux/arm64`), runs as non-root (UID 1001), and uses a read-only root filesystem.

## Quick Start

### With a bearer token

```bash
podman run --rm -p 8080:8080 \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  quay.io/yaacov/kubectl-debug-queries-mcp-server:latest
```

### Skip TLS verification (e.g. development clusters)

```bash
podman run --rm -p 8080:8080 \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  -e MCP_KUBE_INSECURE=true \
  quay.io/yaacov/kubectl-debug-queries-mcp-server:latest
```

### With TLS (HTTPS)

Mount certificate and key files into the container:

```bash
podman run --rm -p 8443:8443 \
  -v ./tls.crt:/tls/tls.crt:ro \
  -v ./tls.key:/tls/tls.key:ro \
  -e MCP_PORT=8443 \
  -e MCP_CERT_FILE=/tls/tls.crt \
  -e MCP_KEY_FILE=/tls/tls.key \
  -e MCP_KUBE_SERVER=https://api.mycluster.example.com:6443 \
  -e MCP_KUBE_TOKEN="$(oc whoami -t)" \
  quay.io/yaacov/kubectl-debug-queries-mcp-server:latest
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_HOST` | `0.0.0.0` | Bind address |
| `MCP_PORT` | `8080` | Listen port |
| `MCP_CERT_FILE` | | TLS certificate path (enables HTTPS) |
| `MCP_KEY_FILE` | | TLS private key path (enables HTTPS) |
| `MCP_KUBE_SERVER` | | Kubernetes API server URL |
| `MCP_KUBE_TOKEN` | | Bearer token for K8s auth |
| `MCP_KUBE_INSECURE` | | Set `true` to skip TLS verification for the K8s API |

## Connecting an AI Client

Once the container is running, the SSE endpoint is available at:

```
http://localhost:8080/sse
```

**Claude Desktop** (add to `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "kubectl-debug-queries": {
      "url": "http://localhost:8080/sse"
    }
  }
}
```

**Cursor IDE:**

Settings → MCP → Add Server → Type: `sse`, URL: `http://localhost:8080/sse`

## Per-Session Authentication via Headers

In SSE mode the container also accepts per-request authentication headers. This lets multiple clients share one server, each with their own credentials:

| Header | Description |
|--------|-------------|
| `Authorization: Bearer <token>` | Bearer token for K8s auth |
| `X-Kubernetes-Server: <url>` | Kubernetes API server URL |

Header values take precedence over environment variables.

## Building the Image

```bash
make vendor                # populate vendor/ (required for container build)
make image-build-amd64     # linux/amd64
make image-build-arm64     # linux/arm64
make image-build-all       # both architectures
```

Push and create a multi-arch manifest:

```bash
make image-push-all        # push arch images + manifest
```

Override the registry and organization:

```bash
make image-build-amd64 IMAGE_REGISTRY=my-registry.example.com IMAGE_ORG=myorg
```

## Docker Compose Example

```yaml
services:
  kubectl-debug-queries:
    image: quay.io/yaacov/kubectl-debug-queries-mcp-server:latest
    ports:
      - "8080:8080"
    environment:
      MCP_KUBE_SERVER: https://api.mycluster.example.com:6443
      MCP_KUBE_TOKEN: ${KUBE_TOKEN}
      MCP_KUBE_INSECURE: "true"
```

Run with:

```bash
KUBE_TOKEN="$(oc whoami -t)" docker compose up
```
