# CLI Usage

kubectl-debug-queries provides subcommands for querying Kubernetes resources, logs, and events from the command line.

All commands use named flags only — no positional arguments.

## Commands

### get

Get a single Kubernetes resource by type, name, and namespace. Columns are auto-detected from the API server (same columns as `kubectl get`).

```bash
kubectl debug-queries get --resource pod --name my-pod --namespace default
kubectl debug-queries get --resource deployment --name nginx --namespace web --format json
kubectl debug-queries get --resource node --name worker-1 --namespace default --format yaml

# Field selection with JSON output
kubectl debug-queries get --resource pod --name my-pod --namespace default --format json \
  --query "select Name, Status"
```

**Flags:**

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--resource` | yes | | Resource type (e.g. pod, deployment, service, configmap, node) |
| `--name` | yes | | Resource name |
| `--namespace` | yes | | Namespace |
| `--format` | no | `table` | Output format: `table`, `markdown`, `json`, `yaml` |
| `--query` / `-q` | no | | TSL query for filtering and field selection (see [Query Language](query-language.md)) |

### list

List Kubernetes resources of a given type. Supports label selectors, sorting by any column, and row limiting.

```bash
# List pods in a namespace
kubectl debug-queries list --resource pods --namespace default

# With label selector and sorting
kubectl debug-queries list --resource pods --namespace kube-system --selector app=nginx --sort-by name

# All namespaces with limit
kubectl debug-queries list --resource deployments --all-namespaces --limit 20

# JSON output
kubectl debug-queries list --resource services --namespace web --format json

# Filter with TSL query ("where" keyword is optional for bare expressions)
kubectl debug-queries list --resource pods --namespace default --query "Status = 'Running'"

# Regex match and sort
kubectl debug-queries list --resource pods --namespace default --query "Name ~= 'nginx-.*' order by Restarts desc"

# JSON output with field selection
kubectl debug-queries list --resource pods --namespace default --format json \
  --query "select Name, Status where Restarts > 0 order by Restarts desc limit 10"
```

**Flags:**

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--resource` | yes | | Resource type (e.g. pods, deployments, services) |
| `--namespace` | yes* | | Namespace (*required unless `--all-namespaces`) |
| `--all-namespaces` | no | `false` | List across all namespaces |
| `--selector` | no | | Label selector (e.g. `app=nginx`, `env in (prod,staging)`) |
| `--sort-by` | no | | Column name to sort by (case-insensitive) |
| `--limit` | no | `0` | Maximum number of rows to return (0 = no limit) |
| `--format` | no | `table` | Output format: `table`, `markdown`, `json`, `yaml` |
| `--query` / `-q` | no | | TSL query for filtering, sorting, and field selection (see [Query Language](query-language.md)) |

### logs

Retrieve logs from a pod or workload. Supports tail lines, time-based filtering, and reverse-time sorting.

The `--name` flag accepts a plain pod name or a `type/name` reference (e.g. `deployment/nginx`). For non-pod resources, a running pod is automatically selected. Supported resource types: `deployment`, `statefulset`, `daemonset`, `replicaset`, `job` (plus common short forms like `deploy`, `sts`, `ds`, `rs`).

By default, log format is auto-detected (JSON, klog, logfmt, CLF) and rendered in a compact smart format that is smaller and more readable than raw output. Unparseable lines pass through with a `[    ]` prefix.

```bash
# Basic pod logs (smart format, auto-detected)
kubectl debug-queries logs --name my-pod --namespace default

# Deployment logs — automatically selects a running pod
kubectl debug-queries logs --name deployment/nginx --namespace default --tail 100

# StatefulSet logs
kubectl debug-queries logs --name statefulset/web --namespace default --tail 50

# Job logs
kubectl debug-queries logs --name job/batch-1 --namespace default

# Last 100 lines, newest first
kubectl debug-queries logs --name my-pod --namespace default --tail 100 --sort-by time_desc

# Specific container, last hour
kubectl debug-queries logs --name my-pod --namespace default --container sidecar --since 1h

# Previous terminated container
kubectl debug-queries logs --name my-pod --namespace default --previous --tail 50

# Raw unprocessed logs
kubectl debug-queries logs --name my-pod --namespace default --tail 100 --format raw

# Parsed JSON output
kubectl debug-queries logs --name my-pod --namespace default --tail 100 --format json

# Filter by log level ("where" is optional for bare expressions)
kubectl debug-queries logs --name deployment/nginx --namespace default --tail 200 \
  --query "level = 'ERROR'"

# Errors and warnings
kubectl debug-queries logs --name my-pod --namespace default --tail 500 \
  --query "level = 'ERROR' or level = 'WARN'"

# Filter by nested field (extra key=value pairs from structured logs)
kubectl debug-queries logs --name deployment/forklift-controller \
  --namespace konveyor-forklift --tail 100 \
  --query "fields.map is not null"

# JSON output with field selection
kubectl debug-queries logs --name my-pod --namespace default --tail 200 --format json \
  --query "select timestamp, level, message where level = 'ERROR'"
```

**Flags:**

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--name` | yes | | Pod name or resource/name (e.g. `my-pod`, `deployment/nginx`) |
| `--namespace` | yes | | Namespace |
| `--container` | no | | Container name (required for multi-container pods) |
| `--previous` | no | `false` | Return logs from previous terminated container |
| `--tail` | no | `0` | Number of lines from the end (0 = all) |
| `--since` | no | | Duration: return logs newer than this (e.g. `1h`, `30m`, `5s`) |
| `--sort-by` | no | `time` | `time` (oldest first) or `time_desc` (newest first) |
| `--format` | no | `smart` | `smart` (auto-detect and compact), `raw`, `json` |
| `--query` / `-q` | no | | TSL query on parsed log fields (see [Query Language](query-language.md)) |

### events

List Kubernetes events, optionally filtered by involved resource.

```bash
# All events in a namespace
kubectl debug-queries events --namespace default

# Filter by resource kind and name
kubectl debug-queries events --namespace default --resource Pod --name my-pod

# All namespaces with sorting and limit
kubectl debug-queries events --all-namespaces --sort-by "last seen" --limit 50

# JSON output
kubectl debug-queries events --namespace web --resource Deployment --format json

# Filter with TSL query ("where" is optional)
kubectl debug-queries events --namespace default --query "Type = 'Warning'"

# BackOff events sorted by last seen
kubectl debug-queries events --namespace default --query "Reason = 'BackOff' order by Last_Seen desc"

# JSON output with field selection
kubectl debug-queries events --namespace default --format json \
  --query "select Reason, Message where Type = 'Warning'"
```

**Flags:**

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--namespace` | yes* | | Namespace (*required unless `--all-namespaces`) |
| `--all-namespaces` | no | `false` | List events across all namespaces |
| `--resource` | no | | Filter by involved object kind (e.g. `Pod`, `Deployment`) |
| `--name` | no | | Filter by involved object name |
| `--sort-by` | no | | Column name to sort by (e.g. `last seen`, `type`, `reason`) |
| `--limit` | no | `0` | Maximum number of rows to return (0 = no limit) |
| `--format` | no | `table` | Output format: `table`, `markdown`, `json`, `yaml` |
| `--query` / `-q` | no | | TSL query for filtering, sorting, and field selection (see [Query Language](query-language.md)) |

## Global Flags

All commands accept standard kubectl flags:

| Flag | Description |
|------|-------------|
| `--kubeconfig` | Path to kubeconfig file |
| `--context` | Kubeconfig context to use |
| `--server` / `-s` | Kubernetes API server URL |
| `--token` | Bearer token for authentication |
| `--namespace` / `-n` | Namespace scope |
| `--insecure-skip-tls-verify` | Skip TLS certificate verification |

## Output Formats

### Resources (get, list, events)

- **table** (default) — Pretty-printed columns with aligned headers, auto-detected from the API server
- **markdown** — GitHub-compatible Markdown table (same columns as `table`)
- **json** — JSON array of row data keyed by column name
- **yaml** — YAML output with one document per row

**Table example:**

```
$ kubectl debug-queries list --resource pods --namespace default
NAME        READY  STATUS   RESTARTS  AGE
my-pod      1/1    Running  0         2d
nginx-abc   1/1    Running  1         5d
```

### Logs

- **smart** (default) — Auto-detects log format (JSON, klog, logfmt, CLF) and renders each line as `[LEVEL] HH:MM:SS source: message key=val`. Typically 50-67% smaller than raw JSON logs. Lines that cannot be parsed pass through with a `[    ]` prefix.
- **raw** — Original unprocessed log text, exactly as returned by the Kubernetes API
- **json** — JSON array of parsed log entries with normalized fields (`timestamp`, `level`, `message`, `source`, `logger`, `fields`)

**Smart format example (JSON logs from a controller):**

```
$ kubectl debug-queries logs --name forklift-controller-abc --namespace konveyor-forklift --tail 5
# format: json, lines: 5
[INFO ] 11:13:26 plan: Reconcile started. plan=migrate-rhel8-nfs ns=mtv-test
[DEBUG] 11:13:26 plan: Skipping reconcile of succeeded plan. plan=migrate-rhel8-nfs
[INFO ] 11:13:26 plan: Reconcile ended. reQ=0
[INFO ] 11:13:26 networkMap: Reconcile ended. reQ=0
[INFO ] 11:13:26 storageMap: Reconcile ended. reQ=0
```

**Smart format example (klog from an API server):**

```
$ kubectl debug-queries logs --name apiserver-abc --namespace openshift-apiserver --tail 3
# format: klog, lines: 3
[INFO ] 11:01:33 policy_source.go:240: refreshing policies
[WARN ] 10:55:33 logging.go:55: [core] grpc: addrConn.createTransport failed to connect
[INFO ] 11:11:33 policy_source.go:240: refreshing policies
```

## Query Language

All commands support an optional `--query` / `-q` flag for TSL (Tree Search Language) filtering. TSL provides SQL-like syntax for filtering, sorting, and field selection.

The `where` keyword is optional -- bare expressions like `Status = 'Running'` are automatically treated as filters.

```bash
# Filter rows (bare expression, "where" auto-prepended)
kubectl debug-queries list --resource pods --namespace default --query "Status = 'Running'"

# Filter + sort + limit
kubectl debug-queries list --resource pods --namespace default \
  --query "Restarts > 0 order by Restarts desc limit 10"

# Field selection (JSON output)
kubectl debug-queries list --resource pods --namespace default --format json --query "select Name, Status"

# Filter parsed log entries by level
kubectl debug-queries logs --name my-pod --namespace default --tail 200 --query "level = 'ERROR'"

# Filter by nested log field
kubectl debug-queries logs --name deployment/forklift-controller \
  --namespace konveyor-forklift --tail 100 --query "fields.map is not null"
```

For the full TSL syntax reference, operators, functions, and examples, see [Query Language (TSL)](query-language.md).
