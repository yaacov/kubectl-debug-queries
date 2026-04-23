# Deployment on OpenShift

Deploy the kubectl-debug-queries MCP server on an OpenShift cluster for use with OpenShift Lightspeed or external AI clients.

## Architecture

```
┌───────────────────┐     ┌───────────────────────────┐     ┌───────────────┐
│  OpenShift        │     │  kubectl-debug-queries    │     │  Kubernetes   │
│  Lightspeed /     │────▶│  MCP Server               │────▶│  API Server   │
│  AI Client        │HTTP │  (openshift-monitoring)   │     │               │
└───────────────────┘     └───────────────────────────┘     └───────────────┘
        │                          │
        │  Authorization:          │  Bearer <user-token>
        │  Bearer <user-token>     │  ──▶ K8s API
        │                          │
        └──────────────────────────┘
```

The MCP server has **no RBAC bindings** of its own. Permissions are determined entirely by the forwarded user token.

## Quick Start

```bash
# Deploy the server
make deploy

# Register with OpenShift Lightspeed
make deploy-olsconfig

# (Optional) Expose externally via Route
make deploy-route
```

## Manifests

All manifests are in the `deploy/` directory:

| File | Resources |
|------|-----------|
| `mcp-server.yaml` | ServiceAccount, Deployment, Service |
| `mcp-route.yaml` | Route (TLS edge termination) |
| `olsconfig-patch.yaml` | OLSConfig patch for Lightspeed |

## Step by Step

### 1. Deploy the Server

```bash
oc apply -f deploy/mcp-server.yaml
```

This creates in `openshift-monitoring`:
- **ServiceAccount** `kubectl-debug-queries-mcp-server`
- **Deployment** running the MCP server in HTTP mode using Streamable HTTP transport on port 8080
- **Service** (ClusterIP) exposing port 8080

The deployment sets `MCP_KUBE_SERVER=https://kubernetes.default.svc` and `MCP_KUBE_INSECURE=true` so the server can reach the in-cluster K8s API.

### 2. Register with OpenShift Lightspeed

```bash
oc patch olsconfig cluster --type merge -p "$(cat deploy/olsconfig-patch.yaml)"
```

This configures Lightspeed to:
- Enable the `MCPServer` feature gate
- Route MCP requests to `http://kubectl-debug-queries-mcp-server.openshift-monitoring.svc:8080/mcp`
- Forward the logged-in user's bearer token via `Authorization: kubernetes`

### 3. (Optional) Expose via Route

```bash
oc apply -f deploy/mcp-route.yaml
```

Creates a Route with TLS edge termination. Get the URL:

```bash
oc get route kubectl-debug-queries-mcp-server -n openshift-monitoring \
  -o jsonpath='https://{.spec.host}/mcp'
```

## Container Image

### Build

```bash
make vendor                # populate vendor/ for container build
make image-build-amd64     # build for linux/amd64
make image-build-arm64     # build for linux/arm64
make image-build-all       # build both
```

### Push

```bash
make image-push-amd64
make image-push-arm64
make image-manifest        # create and push multi-arch manifest
make image-push-all        # push all + manifest
```

### Image Configuration

Override the registry/org/name:

```bash
make image-build-amd64 IMAGE_REGISTRY=my-registry.example.com IMAGE_ORG=myorg
```

### Environment Variables

The container image accepts these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_HOST` | `0.0.0.0` | Bind address |
| `MCP_PORT` | `8080` | Listen port |
| `MCP_CERT_FILE` | | TLS certificate path |
| `MCP_KEY_FILE` | | TLS key path |
| `MCP_KUBE_SERVER` | | K8s API server URL |
| `MCP_KUBE_TOKEN` | | Bearer token |
| `MCP_KUBE_INSECURE` | | Set `true` to skip TLS verify for K8s API |

## Teardown

```bash
make undeploy-all          # remove Lightspeed registration + server
# or individually:
make undeploy-olsconfig    # unregister from Lightspeed
make undeploy-route        # remove route
make undeploy              # remove deployment + service
```

## Security Notes

- The pod runs as non-root (UID 1001) with a read-only root filesystem
- All Linux capabilities are dropped
- seccomp profile is set to `RuntimeDefault`
- No cluster-wide RBAC — all operations use the forwarded user token
- TLS from external clients terminates at the OpenShift router (edge termination)
