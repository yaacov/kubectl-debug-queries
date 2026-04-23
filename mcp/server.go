// Package mcp registers MCP tools for querying Kubernetes resources, logs, and events.
package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yaacov/kubectl-debug-queries/pkg/connection"
	"github.com/yaacov/kubectl-debug-queries/pkg/help"
	"github.com/yaacov/kubectl-debug-queries/pkg/kube"
	ptable "github.com/yaacov/kubectl-debug-queries/pkg/table"
	"github.com/yaacov/kubectl-debug-queries/pkg/version"
	"k8s.io/klog/v2"
)

// DebugReadInput is the input schema for the debug_read tool.
type DebugReadInput struct {
	Command string         `json:"command" jsonschema:"Subcommand: get | list | logs | events"`
	Flags   map[string]any `json:"flags,omitempty" jsonschema:"Command-specific parameters (e.g. resource: 'pod', name: 'my-pod', namespace: 'default')"`
}

// DebugHelpInput is the input schema for the debug_help tool.
type DebugHelpInput struct {
	Command string `json:"command,omitempty" jsonschema:"Subcommand to get help for (e.g. get, list, logs, events). Omit for overview."`
}

// CreateServer creates an MCP server with debug tools registered.
// In HTTP mode the SDK populates req.Extra.Header on every POST with
// that request's HTTP headers, giving each tool call fresh auth credentials.
// In stdio mode there are no HTTP headers and we fall back to CLI defaults.
func CreateServer() *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "kubectl-debug-queries",
		Version: version.Version,
	}, &mcpsdk.ServerOptions{
		Instructions: "Kubernetes resource, log, and event query server. " +
			"Use debug_help to learn available commands and flags. " +
			"Use debug_read with command \"list\" to list resources. " +
			"Use debug_read with command \"get\" to get a specific resource. " +
			"Use debug_read with command \"logs\" to get container logs (supports deployment/name, statefulset/name, etc.). " +
			"Use debug_read with command \"events\" to list events. " +
			"All commands support an optional \"query\" flag for TSL-based filtering. " +
			"IMPORTANT: query fields must match the JSON object structure returned by the command (e.g. \"where status.phase = 'Running'\" for pods, \"where type = 'Warning'\" for events). " +
			"Use output=json first to discover available field paths, then build queries using those paths.",
	})

	registerTools(server)
	return server
}

func registerTools(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "debug_read",
		Description: `Query Kubernetes resources, logs, and events. Use debug_help for flag details.

Subcommands (pass as "command"):
  get     Get a single resource        (flags: resource, name, namespace, output, query)
  list    List resources of a type      (flags: resource, namespace, all_namespaces, selector, sort_by, limit, output, query)
  logs    Retrieve container logs        (flags: name, namespace, container, previous, tail, since, sort_by, output, query)
  events  List Kubernetes events        (flags: namespace, all_namespaces, resource, name, sort_by, limit, output, query)

The "query" flag accepts TSL (Tree Search Language) syntax for filtering.
IMPORTANT: Query field names must match the actual JSON object structure returned by the command.
Use output=json without a query first to discover the available field paths for each resource type.

Common field paths for pods:   name, namespace, status.phase, metadata.labels, spec.nodeName, status.containerStatuses[0].restartCount
Common field paths for events: type, reason, message, count, firstTimestamp, lastTimestamp, involvedObject.kind, involvedObject.name
Shortcut fields: "name" and "namespace" are hoisted from metadata for convenience.

Query examples:
  "where status.phase = 'Running'"                                    Filter pods by phase
  "where name ~= 'nginx-.*' order by metadata.creationTimestamp desc"  Regex match + sort
  "select name, status.phase where status.phase = 'Running'"           Select fields (JSON output)
  "where type = 'Warning'"                                             Filter events by type
  "where level = 'ERROR'"                                              Log levels (always UPPERCASE)
  "where logger ~= 'plan.*'"                                          Filter logs by logger (JSON/zap)
  "where raw_line ~= '.*search-term.*'"                                Full-text log search

Examples:
  {command: "get", flags: {resource: "pod", name: "my-pod", namespace: "default"}}
  {command: "list", flags: {resource: "pods", namespace: "kube-system", query: "where status.phase = 'Running'"}}
  {command: "list", flags: {resource: "pods", namespace: "default", output: "json", query: "select name, status.phase"}}
  # Discover log field names — always start here
  {command: "logs", flags: {name: "deployment/my-app", namespace: "ns", tail: 5, output: "json"}}
  {command: "logs", flags: {name: "deployment/my-app", namespace: "ns", tail: 100, query: "where level = 'ERROR'"}}
  {command: "logs", flags: {name: "deployment/my-app", namespace: "ns", container: "sidecar", tail: 50}}
  {command: "logs", flags: {name: "my-pod", namespace: "ns", output: "json", query: "select timestamp, level, message where level = 'ERROR'"}}
  {command: "events", flags: {namespace: "default", query: "where type = 'Warning'"}}`,
	}, handleDebugRead)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "debug_help",
		Description: `Get detailed help for debug_read subcommands.

WHEN TO USE: Before calling debug_read, call debug_help("<command>") to learn the
available flags and their meaning.

Commands: get, list, logs, events
Omit command for an overview of all subcommands.`,
	}, handleDebugHelp)
}

func handleDebugRead(ctx context.Context, req *mcpsdk.CallToolRequest, input DebugReadInput) (*mcpsdk.CallToolResult, any, error) {
	if req.Extra != nil && req.Extra.Header != nil {
		ctx = connection.WithCredsFromHeaders(ctx, req.Extra.Header)
	}

	cfg := connection.ResolveRESTConfig(ctx)
	if cfg == nil {
		return textResult("Kubernetes credentials not configured. Provide them via --kubeconfig, --token flag, or Authorization header."), nil, nil
	}

	command := strings.TrimSpace(strings.ToLower(input.Command))
	if command == "" {
		return textResult("Missing required field 'command'. Use one of: get, list, logs, events.\nCall debug_help for details."), nil, nil
	}

	flags := input.Flags
	if flags == nil {
		flags = map[string]any{}
	}

	if kube.FlagStr(flags, "output") == "" {
		switch command {
		case "get", "list", "events":
			flags["output"] = "markdown"
		}
	}

	clients, err := kube.NewClients(cfg)
	if err != nil {
		return textResult(fmt.Sprintf("Failed to create Kubernetes client: %v", err)), nil, nil
	}

	t0 := time.Now()
	var result string

	switch command {
	case "get":
		result, err = kube.Get(ctx, clients,
			kube.FlagStr(flags, "resource"),
			kube.FlagStr(flags, "name"),
			kube.FlagStr(flags, "namespace"),
			kube.FlagStr(flags, "output"),
			kube.FlagStr(flags, "query"))
	case "list":
		result, err = kube.List(ctx, clients,
			kube.FlagStr(flags, "resource"),
			kube.FlagStr(flags, "namespace"),
			kube.FlagStr(flags, "selector"),
			kube.FlagStr(flags, "sort_by"),
			kube.FlagInt(flags, "limit"),
			kube.FlagBool(flags, "all_namespaces"),
			kube.FlagStr(flags, "output"),
			kube.FlagStr(flags, "query"))
	case "logs":
		result, err = kube.Logs(ctx, clients,
			kube.FlagStr(flags, "name"),
			kube.FlagStr(flags, "namespace"),
			kube.FlagStr(flags, "container"),
			kube.FlagBool(flags, "previous"),
			kube.FlagInt(flags, "tail"),
			kube.FlagStr(flags, "since"),
			kube.FlagStr(flags, "sort_by"),
			kube.FlagStr(flags, "output"),
			kube.FlagStr(flags, "query"))
	case "events":
		result, err = kube.Events(ctx, clients,
			kube.FlagStr(flags, "namespace"),
			kube.FlagStr(flags, "resource"),
			kube.FlagStr(flags, "name"),
			kube.FlagStr(flags, "sort_by"),
			kube.FlagInt(flags, "limit"),
			kube.FlagBool(flags, "all_namespaces"),
			kube.FlagStr(flags, "output"),
			kube.FlagStr(flags, "query"))
	default:
		return textResult(fmt.Sprintf("Unknown command %q. Available: get, list, logs, events.\nCall debug_help(\"%s\") for details.", command, command)), nil, nil
	}

	if err != nil {
		format := kube.FlagStr(flags, "output")
		if kube.IsJSONFormat(format) {
			return textResult(kube.JSONError(err)), nil, nil
		}
		return textResult(friendlyError(command, err)), nil, nil
	}

	klog.V(1).Infof("debug_read %s completed in %.3fs", command, time.Since(t0).Seconds())
	if result == "" && kube.IsJSONFormat(kube.FlagStr(flags, "output")) {
		return textResult(kube.JSONEmpty), nil, nil
	}
	return textResult(result), nil, nil
}

func handleDebugHelp(_ context.Context, _ *mcpsdk.CallToolRequest, input DebugHelpInput) (*mcpsdk.CallToolResult, any, error) {
	command := strings.TrimSpace(strings.ToLower(input.Command))
	return textResult(help.GenerateHelp(command)), nil, nil
}

func textResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}
}

func friendlyError(command string, err error) string {
	errStr := err.Error()

	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		return "Connection failed: cannot reach Kubernetes API server.\nCheck that --server is correct and the server is reachable."
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return "Request timed out querying the Kubernetes API.\nThe server may be overloaded."
	}
	if strings.Contains(errStr, "HTTP 401") || strings.Contains(errStr, "Unauthorized") {
		return "Authentication failed (HTTP 401). Ensure a valid bearer token is provided via --token flag or Authorization header."
	}
	if strings.Contains(errStr, "HTTP 403") || strings.Contains(errStr, "forbidden") {
		return "Authorization denied (HTTP 403). The token may lack permissions for this operation."
	}

	return fmt.Sprintf("Error in %s: %s", command, errStr)
}

// TableOptions builds table.Options from a flags map.
func TableOptions(flags map[string]any) ptable.Options {
	return ptable.Options{
		SortBy: kube.FlagStr(flags, "sort_by"),
		Limit:  kube.FlagInt(flags, "limit"),
	}
}
