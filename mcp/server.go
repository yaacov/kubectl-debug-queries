// Package mcp registers MCP tools for querying Kubernetes resources, logs, and events.
package mcp

import (
	"context"
	"fmt"
	"net/http"
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
func CreateServer(capturedHeaders http.Header) *mcpsdk.Server {
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
			"All commands support an optional \"query\" flag for TSL-based filtering (e.g. \"where Status = 'Running'\").",
	})

	registerTools(server, capturedHeaders)
	return server
}

func registerTools(server *mcpsdk.Server, capturedHeaders http.Header) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "debug_read",
		Description: `Query Kubernetes resources, logs, and events. Use debug_help for flag details.

Subcommands (pass as "command"):
  get     Get a single resource        (flags: resource, name, namespace, output, query)
  list    List resources of a type      (flags: resource, namespace, all_namespaces, selector, sort_by, limit, output, query)
  logs    Retrieve container logs        (flags: name, namespace, container, previous, tail, since, sort_by, output, query)
  events  List Kubernetes events        (flags: namespace, all_namespaces, resource, name, sort_by, limit, output, query)

The "query" flag accepts TSL (Tree Search Language) syntax for filtering:
  "where Status = 'Running'"
  "where Name ~= 'nginx-.*' order by Age desc limit 10"
  "select Name, Status where Restarts > 5"

For JSON output, SELECT controls which fields appear. For table output, columns are unchanged.

Examples:
  {command: "get", flags: {resource: "pod", name: "my-pod", namespace: "default"}}
  {command: "list", flags: {resource: "pods", namespace: "kube-system", query: "where Status = 'Running'"}}
  {command: "list", flags: {resource: "pods", namespace: "default", output: "json", query: "select Name, Status where Restarts > 0"}}
  {command: "logs", flags: {name: "my-pod", namespace: "default", tail: 100, query: "where level = 'ERROR'"}}
  {command: "events", flags: {namespace: "default", query: "where Type = 'Warning'"}}`,
	}, wrapWithHeaders(handleDebugRead, capturedHeaders))

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name: "debug_help",
		Description: `Get detailed help for debug_read subcommands.

WHEN TO USE: Before calling debug_read, call debug_help("<command>") to learn the
available flags and their meaning.

Commands: get, list, logs, events
Omit command for an overview of all subcommands.`,
	}, wrapWithHeaders(handleDebugHelp, capturedHeaders))
}

func handleDebugRead(ctx context.Context, req *mcpsdk.CallToolRequest, input DebugReadInput) (*mcpsdk.CallToolResult, struct{}, error) {
	if req.Extra != nil && req.Extra.Header != nil {
		ctx = connection.WithCredsFromHeaders(ctx, req.Extra.Header)
	}

	cfg := connection.ResolveRESTConfig(ctx)
	if cfg == nil {
		return textResult("Kubernetes credentials not configured. Provide them via --kubeconfig, --token flag, or Authorization header."), struct{}{}, nil
	}

	command := strings.TrimSpace(strings.ToLower(input.Command))
	if command == "" {
		return textResult("Missing required field 'command'. Use one of: get, list, logs, events.\nCall debug_help for details."), struct{}{}, nil
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
		return textResult(fmt.Sprintf("Failed to create Kubernetes client: %v", err)), struct{}{}, nil
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
		return textResult(fmt.Sprintf("Unknown command %q. Available: get, list, logs, events.\nCall debug_help(\"%s\") for details.", command, command)), struct{}{}, nil
	}

	if err != nil {
		format := kube.FlagStr(flags, "output")
		if kube.IsJSONFormat(format) {
			return textResult(kube.JSONError(err)), struct{}{}, nil
		}
		return textResult(friendlyError(command, err)), struct{}{}, nil
	}

	klog.V(1).Infof("debug_read %s completed in %.3fs", command, time.Since(t0).Seconds())
	if result == "" && kube.IsJSONFormat(kube.FlagStr(flags, "output")) {
		return textResult(kube.JSONEmpty), struct{}{}, nil
	}
	return textResult(result), struct{}{}, nil
}

func handleDebugHelp(_ context.Context, _ *mcpsdk.CallToolRequest, input DebugHelpInput) (*mcpsdk.CallToolResult, struct{}, error) {
	command := strings.TrimSpace(strings.ToLower(input.Command))
	return textResult(help.GenerateHelp(command)), struct{}{}, nil
}

func textResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
	}
}

func wrapWithHeaders[In, Out any](
	handler func(context.Context, *mcpsdk.CallToolRequest, In) (*mcpsdk.CallToolResult, Out, error),
	headers http.Header,
) func(context.Context, *mcpsdk.CallToolRequest, In) (*mcpsdk.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest, input In) (*mcpsdk.CallToolResult, Out, error) {
		if req.Extra == nil && headers != nil {
			req.Extra = &mcpsdk.RequestExtra{Header: headers}
		} else if req.Extra != nil && req.Extra.Header == nil && headers != nil {
			req.Extra.Header = headers
		}
		return handler(ctx, req, input)
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
