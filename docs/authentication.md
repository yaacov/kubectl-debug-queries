# Authentication

kubectl-debug-queries uses `k8s.io/client-go` for authentication, supporting all standard kubeconfig methods.

## Supported Auth Methods

| Method | How It Works |
|--------|-------------|
| **Bearer token** | Inline token in kubeconfig (e.g. `oc login`) — used directly |
| **Bearer token file** | Token read from file path in kubeconfig |
| **Exec provider** | External command produces a token (e.g. `oc`, `gcloud`, `aws-iam-authenticator`) |
| **Client certificates** | Mutual TLS — works directly for K8s API |
| **OIDC / Auth provider** | Handled by client-go auth plugins |

## How It Works

### Standard Flow (bearer token, exec)

1. `client-go` loads the kubeconfig and resolves credentials
2. A `rest.Config` is built with the credentials
3. The config is used to create Kubernetes clients for resource queries

### Client Certificate Flow

Client certificates work directly with the Kubernetes API. The tool builds an insecure transport (skipping CA verification for self-signed certs) and uses the client cert credentials for all API calls.

## CLI Flag Overrides

Standard kubectl flags override kubeconfig values:

```bash
# Explicit token
kubectl debug-queries list --resource pods --namespace default --token sha256~xxxxx

# Explicit server + token
kubectl debug-queries list --resource pods --namespace default \
  --server https://api.cluster.example.com:6443 --token sha256~xxxxx

# Different kubeconfig or context
kubectl debug-queries list --resource pods --namespace default \
  --kubeconfig /path/to/config --context my-cluster
```

## SSE Mode (MCP Server)

In SSE mode, per-session credentials can be provided via HTTP headers, which take highest priority:

```
Authorization: Bearer <token>
X-Kubernetes-Server: https://api.cluster.example.com:6443
```

This allows a single MCP server instance to serve multiple users, each authenticated with their own token (e.g. OpenShift Lightspeed forwarding the logged-in user's token).

## Credential Precedence

Three tiers (highest priority first):

1. **SSE HTTP headers** (per-session) — `Authorization: Bearer <token>`, `X-Kubernetes-Server: <url>`
2. **CLI defaults** — kubeconfig flags (`--kubeconfig`, `--context`, `--token`, `--server`)
3. **Fallback** — default kubeconfig from `$KUBECONFIG` or `~/.kube/config`

## Debugging Authentication

Use klog verbosity to see which auth method is detected:

```bash
kubectl debug-queries list --resource pods --namespace default --v=2
```

Output includes:

```
[auth] API server: https://api.cluster.example.com:6443
[auth] Method: bearer token (length 64)
```
