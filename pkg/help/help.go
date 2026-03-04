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
  output     (optional)  Output format: table (default), markdown, json, yaml
  query      (optional)  TSL query for field selection (e.g. "select Name, Status")

Query examples:
  "select Name, Status"                     Select specific fields (JSON/YAML output)

Examples:
  {command: "get", flags: {resource: "pod", name: "my-pod", namespace: "default"}}
  {command: "get", flags: {resource: "deployment", name: "nginx", namespace: "web", output: "json"}}
  {command: "get", flags: {resource: "pod", name: "my-pod", namespace: "default", output: "json", query: "select Name, Status"}}`

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
  output          (optional)  Output format: table (default), markdown, json, yaml
  query           (optional)  TSL query for filtering, sorting, and field selection

Query syntax (TSL — Tree Search Language):
  Supports: SELECT, WHERE, ORDER BY / SORT BY, LIMIT
  Operators: =, !=, <, >, <=, >=, like, ilike, ~= (regex), in, between, is null, is not null
  Logic: and, or, not
  Column names with spaces use underscores: Last_Seen, Nominated_Node

Query examples:
  "where Status = 'Running'"                          Filter by column value
  "where Name ~= 'nginx-.*'"                          Regex match
  "where Restarts > 5 and Status != 'Running'"        Combined conditions
  "where Status = 'Running' order by Name"             Filter and sort
  "where Status = 'Running' order by Age desc limit 10"  Filter, sort descending, limit
  "select Name, Status where Restarts > 0"             Select fields (JSON/YAML output only)

Note: For table/markdown output, columns are always the server-side defaults.
      SELECT only affects JSON and YAML output.

Examples:
  {command: "list", flags: {resource: "pods", namespace: "default"}}
  {command: "list", flags: {resource: "pods", namespace: "kube-system", selector: "app=nginx", sort_by: "name"}}
  {command: "list", flags: {resource: "deployments", all_namespaces: true, limit: 20}}
  {command: "list", flags: {resource: "pods", namespace: "default", query: "where Status = 'Running'"}}
  {command: "list", flags: {resource: "pods", namespace: "default", output: "json", query: "select Name, Status where Restarts > 0"}}`

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

Queryable log fields: timestamp, level, message, source, logger, raw_line, format, parsed
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
  output          (optional)  Output format: table (default), markdown, json, yaml
  query           (optional)  TSL query for filtering, sorting, and field selection

Query examples:
  "where Type = 'Warning'"                             Filter by event type
  "where Reason = 'BackOff'"                           Filter by reason
  "where Type = 'Warning' order by Last_Seen desc"     Filter and sort
  "select Reason, Message where Type = 'Warning'"      Select fields (JSON/YAML output)

Note: Column names with spaces use underscores in queries: Last_Seen, First_Seen.

Examples:
  {command: "events", flags: {namespace: "default"}}
  {command: "events", flags: {namespace: "default", resource: "Pod", name: "my-pod"}}
  {command: "events", flags: {namespace: "default", query: "where Type = 'Warning'"}}
  {command: "events", flags: {namespace: "default", output: "json", query: "select Reason, Message where Type = 'Warning'"}}`

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
			"All commands support an optional \"query\" flag using TSL (Tree Search Language) syntax:",
			"  WHERE filtering:    \"where Status = 'Running'\"",
			"  Regex matching:     \"where Name ~= 'nginx-.*'\"",
			"  Sorting:            \"where Status = 'Running' order by Name desc\"",
			"  Limiting:           \"where Status = 'Running' limit 10\"",
			"  Field selection:    \"select Name, Status where Restarts > 0\" (JSON/YAML output)",
			"",
			"Call debug_help(\"<command>\") for detailed flag descriptions and examples.",
			"",
			"QUICK EXAMPLES:",
			`  {command: "get", flags: {resource: "pod", name: "my-pod", namespace: "default"}}`,
			`  {command: "list", flags: {resource: "pods", namespace: "kube-system", query: "where Status = 'Running'"}}`,
			`  {command: "logs", flags: {name: "my-pod", namespace: "default", tail: 100, query: "where level = 'ERROR'"}}`,
			`  {command: "events", flags: {namespace: "default", query: "where Type = 'Warning'"}}`,
		}
		return strings.Join(lines, "\n")
	}
}
