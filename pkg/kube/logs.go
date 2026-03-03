package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/yaacov/kubectl-debug-queries/pkg/logparse"
	"github.com/yaacov/kubectl-debug-queries/pkg/query"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// Logs retrieves logs with the given options.
// The name parameter can be a plain pod name ("my-pod") or a resource reference
// ("deployment/nginx", "statefulset/web", "job/batch-1"). For non-pod resources,
// the workload's label selector is used to find a running pod.
// The format parameter controls output: "" (default) = smart compact, "raw" = unprocessed, "json" = parsed JSON array.
// The queryStr applies TSL-based filtering to parsed log entries (smart and json formats).
func Logs(ctx context.Context, clients *Clients, name, namespace, container string, previous bool, tail int, since string, sortBy string, format string, queryStr string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	podName, err := ResolvePodName(ctx, clients, name, namespace)
	if err != nil {
		return "", err
	}

	if container == "" {
		container, err = ResolveContainer(ctx, clients, podName, namespace)
		if err != nil {
			return "", err
		}
	}

	opts := &corev1.PodLogOptions{
		Container: container,
		Previous:  previous,
	}

	if tail > 0 {
		t := int64(tail)
		opts.TailLines = &t
	}

	if since != "" {
		d, err := time.ParseDuration(since)
		if err != nil {
			return "", fmt.Errorf("invalid since duration %q: %w", since, err)
		}
		secs := int64(d.Seconds())
		opts.SinceSeconds = &secs
	}

	klog.V(2).Infof("[logs] fetching logs for %s/%s (container=%s, tail=%d, since=%s)", namespace, podName, container, tail, since)

	req := clients.Clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("streaming logs: %w", err)
	}
	defer stream.Close()

	body, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("reading logs: %w", err)
	}

	result := string(body)

	if strings.EqualFold(sortBy, "time_desc") {
		result = reverseLines(result)
	}

	switch strings.ToLower(format) {
	case "raw":
		return result, nil
	case "json":
		return formatLogsJSON(result, queryStr)
	default:
		return formatLogsSmart(result, queryStr)
	}
}

// formatLogsJSON parses log lines, applies the query, and returns JSON output.
// SELECT projects fields in the JSON output.
func formatLogsJSON(text string, queryStr string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return JSONEmpty, nil
	}

	entries, _ := logparse.ParseLines(text)
	items := logEntriesToMaps(entries)

	queryOpts, err := query.ParseQueryString(queryStr)
	if err != nil {
		return "", fmt.Errorf("invalid query: %w", err)
	}

	items, err = query.ApplyQuery(items, queryOpts)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}

	if len(items) == 0 {
		return JSONEmpty, nil
	}

	for _, item := range items {
		delete(item, "_index")
	}

	if queryOpts.HasSelect {
		projected := make([]map[string]interface{}, len(items))
		for i, item := range items {
			p := make(map[string]interface{}, len(queryOpts.Select))
			for _, sel := range queryOpts.Select {
				if v, err := query.GetValue(item, sel.Alias, queryOpts.Select); err == nil && v != nil {
					p[sel.Alias] = v
				}
			}
			projected[i] = p
		}
		items = projected
	}

	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling log entries: %w", err)
	}
	return string(b), nil
}

// formatLogsSmart parses log lines, applies WHERE filtering from the query,
// and renders in compact smart format.
func formatLogsSmart(text string, queryStr string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil
	}

	entries, det := logparse.ParseLines(text)
	if det.Format == logparse.FormatPlain || len(entries) == 0 {
		return text, nil
	}

	if queryStr != "" {
		items := logEntriesToMaps(entries)

		queryOpts, err := query.ParseQueryString(queryStr)
		if err != nil {
			return "", fmt.Errorf("invalid query: %w", err)
		}

		items, err = query.ApplyQuery(items, queryOpts)
		if err != nil {
			return "", fmt.Errorf("query error: %w", err)
		}

		filtered := make([]logparse.LogEntry, 0, len(items))
		for _, item := range items {
			if idx, ok := item["_index"]; ok {
				if i, ok := idx.(int); ok && i < len(entries) {
					filtered = append(filtered, entries[i])
				}
			}
		}
		entries = filtered
	}

	if len(entries) == 0 {
		return "(no matching log entries)", nil
	}

	header := fmt.Sprintf("# format: %s, lines: %d", det.Format, len(entries))
	body := logparse.RenderSmart(entries)
	return header + "\n" + body, nil
}

// logEntriesToMaps converts parsed log entries to maps for query processing.
// Each map includes an _index field for mapping back to the original entry.
func logEntriesToMaps(entries []logparse.LogEntry) []map[string]interface{} {
	items := make([]map[string]interface{}, len(entries))
	for i, e := range entries {
		item := map[string]interface{}{
			"_index":    i,
			"timestamp": e.Timestamp,
			"level":     e.Level,
			"message":   e.Message,
			"source":    e.Source,
			"logger":    e.Logger,
			"raw_line":  e.RawLine,
			"format":    e.Format.String(),
			"parsed":    e.Parsed,
		}
		if len(e.Fields) > 0 {
			fieldsMap := make(map[string]interface{}, len(e.Fields))
			for k, v := range e.Fields {
				fieldsMap[k] = v
			}
			item["fields"] = fieldsMap
		}
		items[i] = item
	}
	return items
}

func reverseLines(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return strings.Join(lines, "\n")
}
