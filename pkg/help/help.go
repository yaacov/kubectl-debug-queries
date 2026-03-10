// Package help provides help text for debug commands.
package help

import "strings"

// GenerateHelp returns help text for the given command or topic.
// Pass an empty string for an overview of all commands.
func GenerateHelp(command string) string {
	switch command {
	case "get":
		return `debug_read "get" — Retrieve a single Kubernetes resource

Returns the resource in server-side table format (same columns as kubectl get).

Flags:
  resource   (required)  Resource type (e.g. pod, deployment, service, configmap, node, virtualmachine)
  name       (required)  Resource name
  namespace  (required)  Namespace
  output     (optional)  Output format: markdown (default), table, json, yaml
  query      (optional)  TSL query for field selection (e.g. "select name, status.phase")

IMPORTANT: Query fields must match the JSON object structure (use output=json to discover fields).
Shortcut fields: "name" and "namespace" are hoisted from metadata for convenience.

Query examples (pod fields):
  "select name, status.phase"               Select specific fields (JSON/YAML output)
  "select name, spec.nodeName"              Select node placement

Examples:
  {command: "get", flags: {resource: "pod", name: "my-pod", namespace: "default"}}
  {command: "get", flags: {resource: "deployment", name: "nginx", namespace: "web", output: "json"}}
  {command: "get", flags: {resource: "pod", name: "my-pod", namespace: "default", output: "json", query: "select name, status.phase"}}`

	case "list":
		return `debug_read "list" — List Kubernetes resources

Returns resources in server-side table format with auto-detected columns.
Supports label selectors, sorting by any column, and row limiting.

Flags:
  resource        (required)  Resource type (e.g. pods, deployments, services, nodes)
  namespace       (required)  Namespace (required unless all_namespaces is true)
  all_namespaces  (optional)  Boolean. List across all namespaces (overrides namespace)
  selector        (optional)  Label selector (e.g. "app=nginx", "env in (prod,staging)")
  sort_by         (optional)  Column name to sort by (case-insensitive, e.g. "name", "age", "status")
  limit           (optional)  Maximum number of rows to return
  output          (optional)  Output format: markdown (default), table, json, yaml
  query           (optional)  TSL query for filtering, sorting, and field selection

Query syntax (TSL — Tree Search Language):
  Supports: SELECT, WHERE, ORDER BY / SORT BY, LIMIT
  Operators: =, !=, <, >, <=, >=, like, ilike, ~= (regex), in, between, is null, is not null
  Logic: and, or, not

IMPORTANT: Query fields must match the JSON object structure returned by the Kubernetes API,
NOT the table column names. Use output=json without a query to discover field paths.
Shortcut fields: "name" and "namespace" are hoisted from metadata for convenience.

Common field paths for pods:
  name, namespace, status.phase, spec.nodeName, metadata.labels,
  status.containerStatuses[0].restartCount, metadata.creationTimestamp

Query examples (for pods):
  "where status.phase = 'Running'"                                        Filter by phase
  "where name ~= 'nginx-.*'"                                              Regex match on name
  "where status.containerStatuses[0].restartCount > 5"                     Filter by restarts
  "where status.phase = 'Running' order by name"                           Filter and sort
  "where status.phase = 'Running' order by metadata.creationTimestamp desc limit 10"
  "select name, status.phase where status.phase != 'Running'"              Select fields (JSON/YAML only)

Note: For table/markdown output, columns are always the server-side defaults.
      SELECT only affects JSON and YAML output.
      The sort_by flag sorts by table column names; ORDER BY in queries sorts by JSON field paths.

Examples:
  {command: "list", flags: {resource: "pods", namespace: "default"}}
  {command: "list", flags: {resource: "pods", namespace: "kube-system", selector: "app=nginx", sort_by: "name"}}
  {command: "list", flags: {resource: "deployments", all_namespaces: true, limit: 20}}
  {command: "list", flags: {resource: "pods", namespace: "default", query: "where status.phase = 'Running'"}}
  {command: "list", flags: {resource: "pods", namespace: "default", output: "json", query: "select name, status.phase"}}`

	case "logs":
		return `debug_read "logs" — Retrieve container logs

Returns log lines from a pod or workload. Supports tail, since, previous container,
and reverse-time sorting.

The name can be a plain pod name ("my-pod") or a resource reference
("deployment/nginx", "statefulset/web", "daemonset/agent", "replicaset/web-abc",
"job/batch-1"). For non-pod resources, a running pod is automatically selected.

By default, logs are auto-detected (JSON, klog, logfmt, CLF) and rendered in a
compact format: [LEVEL] HH:MM:SS source: message key=val. This is typically smaller
than raw output. Unparseable lines pass through with a [    ] prefix.

Flags:
  name       (required)  Pod name or resource/name (e.g. "my-pod", "deployment/nginx")
  namespace  (required)  Namespace
  container  (optional)  Container name (required for multi-container pods)
  previous   (optional)  Boolean. Return logs from the previous terminated container
  tail       (optional)  Number of lines from the end of the log to return
  since      (optional)  Duration string (e.g. "1h", "30m", "5s") — return logs newer than this
  sort_by    (optional)  "time" (default, oldest first) or "time_desc" (newest first)
  output     (optional)  Output format: smart (default), raw, json
                          smart  Auto-detect and render compact (default)
                          raw    Original unprocessed log text
                          json   JSON array of parsed log entries
  query      (optional)  TSL query on parsed log fields (smart and json formats)

Query fields come from the parsed log entry JSON structure:
  timestamp, level, message, source, logger, raw_line, format, parsed
  Nested fields: fields.<key> (e.g. fields.request_id)

Query examples:
  "where level = 'ERROR'"                              Filter by log level
  "where message ~= 'timeout'"                         Regex match on message
  "where level = 'ERROR' or level = 'WARN'"            Multiple levels
  "select timestamp, level, message where level = 'ERROR'"  Select fields (JSON only)

Examples:
  {command: "logs", flags: {name: "my-pod", namespace: "default"}}
  {command: "logs", flags: {name: "deployment/nginx", namespace: "default", tail: 100}}
  {command: "logs", flags: {name: "my-pod", namespace: "default", tail: 200, query: "where level = 'ERROR'"}}
  {command: "logs", flags: {name: "my-pod", namespace: "default", output: "json", query: "select timestamp, level, message where level = 'ERROR'"}}
  {command: "logs", flags: {name: "my-pod", namespace: "default", previous: true, tail: 50}}`

	case "events":
		return `debug_read "events" — List Kubernetes events

Returns events in server-side table format. Optionally filter by involved resource.

Flags:
  namespace       (required)  Namespace (required unless all_namespaces is true)
  all_namespaces  (optional)  Boolean. List events across all namespaces
  resource        (optional)  Filter by involved object kind (e.g. "Pod", "Deployment")
  name            (optional)  Filter by involved object name
  sort_by         (optional)  Column name to sort by (e.g. "last seen", "type", "reason")
  limit           (optional)  Maximum number of rows to return
  output          (optional)  Output format: markdown (default), table, json, yaml
  query           (optional)  TSL query for filtering, sorting, and field selection

IMPORTANT: Query fields must match the JSON object structure of Kubernetes Event objects,
NOT the table column names. Use output=json without a query to discover field paths.

Event JSON fields: type, reason, message, count, firstTimestamp, lastTimestamp,
  involvedObject.kind, involvedObject.name, source.component, metadata.name, metadata.namespace

Query examples:
  "where type = 'Warning'"                                  Filter by event type
  "where reason = 'BackOff'"                                Filter by reason
  "where type = 'Warning' order by lastTimestamp desc"      Filter and sort by time
  "select reason, message where type = 'Warning'"           Select fields (JSON/YAML output)

Note: The sort_by flag sorts by table column names; ORDER BY in queries sorts by JSON field paths.

Examples:
  {command: "events", flags: {namespace: "default"}}
  {command: "events", flags: {namespace: "default", resource: "Pod", name: "my-pod"}}
  {command: "events", flags: {namespace: "default", query: "where type = 'Warning'"}}
  {command: "events", flags: {namespace: "default", output: "json", query: "select reason, message where type = 'Warning'"}}`

	default:
		lines := []string{
			"debug_read — Query Kubernetes resources, logs, and events",
			"",
			"SUBCOMMANDS (pass as \"command\"):",
			"  get     Get a single resource by name             (flags: resource, name, namespace, output, query)",
			"  list    List resources of a type                   (flags: resource, namespace, all_namespaces, selector, sort_by, limit, output, query)",
			"  logs    Retrieve container logs                     (flags: name, namespace, container, previous, tail, since, sort_by, output, query)",
			"  events  List Kubernetes events                     (flags: namespace, all_namespaces, resource, name, sort_by, limit, output, query)",
			"",
			"All commands support an optional \"query\" flag using TSL (Tree Search Language) syntax.",
			"IMPORTANT: Query fields must match the JSON object structure returned by the command.",
			"Use output=json without a query to discover the available field paths for each resource type.",
			"Shortcut fields: \"name\" and \"namespace\" are hoisted from metadata for convenience.",
			"",
			"Query examples:",
			"  WHERE filtering:    \"where status.phase = 'Running'\"                          (pods)",
			"  Regex matching:     \"where name ~= 'nginx-.*'\"",
			"  Sorting:            \"where status.phase = 'Running' order by name desc\"",
			"  Limiting:           \"where status.phase = 'Running' limit 10\"",
			"  Field selection:    \"select name, status.phase\" (JSON/YAML output)",
			"  Events:             \"where type = 'Warning'\"",
			"  Logs:               \"where level = 'ERROR'\"",
			"",
			"Call debug_help(\"<command>\") for detailed flag descriptions and examples.",
			"",
			"QUICK EXAMPLES:",
			`  {command: "get", flags: {resource: "pod", name: "my-pod", namespace: "default"}}`,
			`  {command: "list", flags: {resource: "pods", namespace: "kube-system", query: "where status.phase = 'Running'"}}`,
			`  {command: "logs", flags: {name: "my-pod", namespace: "default", tail: 100, query: "where level = 'ERROR'"}}`,
			`  {command: "events", flags: {namespace: "default", query: "where type = 'Warning'"}}`,
		}
		return strings.Join(lines, "\n")
	}
}
