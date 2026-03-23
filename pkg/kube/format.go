package kube

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaacov/kubectl-debug-queries/pkg/query"
	ptable "github.com/yaacov/kubectl-debug-queries/pkg/table"
	sigyaml "sigs.k8s.io/yaml"
)

const cellsKey = "_columns"

// Synthetic top-level keys hoisted from metadata for convenient querying.
var syntheticKeys = []string{"name", "namespace"}

// FormatTable converts a ServerTable to a rendered string in the given format.
// Queries (WHERE, ORDER BY, LIMIT) always run against the full Kubernetes object.
// Output projection depends on format:
//   - table/markdown: always project to server-side columns
//   - json/yaml without SELECT: output the full object
//   - json/yaml with SELECT: output only the selected fields
func FormatTable(tbl *ServerTable, format string, opts ptable.Options, queryStr string, allNamespaces bool) (string, error) {
	if format == "" {
		format = "markdown"
	}

	queryOpts, err := query.ParseQueryString(queryStr)
	if err != nil {
		return "", fmt.Errorf("invalid query: %w", err)
	}

	columnNames := make([]string, len(tbl.Columns))
	for i, c := range tbl.Columns {
		columnNames[i] = c.Name
	}

	items := rowsToFullItems(tbl)

	items, err = query.ApplyQuery(items, queryOpts)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}

	switch format {
	case "json":
		items = applyLimit(items, opts.Limit)
		return formatJSONItems(items, queryOpts)
	case "yaml":
		items = applyLimit(items, opts.Limit)
		return formatYAMLItems(items, queryOpts)
	case "markdown":
		opts.Markdown = true
		if allNamespaces {
			columnNames, items = injectNamespaceColumn(columnNames, items)
		}
		return renderFilteredTable(items, columnNames, opts), nil
	default:
		if allNamespaces {
			columnNames, items = injectNamespaceColumn(columnNames, items)
		}
		return renderFilteredTable(items, columnNames, opts), nil
	}
}

// rowsToFullItems extracts the full Kubernetes object from each row for query
// filtering. Server-side column values are embedded under the _columns key so
// that table/markdown rendering can project back to them after filtering.
// Synthetic "name" and "namespace" keys are hoisted from metadata so users can
// write "where name = 'foo'" instead of "where metadata.name = 'foo'".
func rowsToFullItems(tbl *ServerTable) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(tbl.Rows))
	for _, r := range tbl.Rows {
		item := make(map[string]interface{})
		for k, v := range r.Object {
			item[k] = v
		}

		if md, ok := item["metadata"].(map[string]interface{}); ok {
			for _, key := range syntheticKeys {
				if v, exists := md[key]; exists {
					item[key] = v
				}
			}
		}

		cells := make(map[string]interface{}, len(tbl.Columns))
		for j, col := range tbl.Columns {
			if j < len(r.Cells) {
				cells[col.Name] = r.Cells[j]
			}
		}
		item[cellsKey] = cells

		items = append(items, item)
	}
	return items
}

// stripInternalKeys returns a shallow copy of item without the internal
// _columns key or synthetic shortcut keys (name, namespace).
func stripInternalKeys(item map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(item))
	for k, v := range item {
		if k == cellsKey {
			continue
		}
		skip := false
		for _, sk := range syntheticKeys {
			if k == sk {
				skip = true
				break
			}
		}
		if !skip {
			out[k] = v
		}
	}
	return out
}

// injectNamespaceColumn inserts a "Namespace" column right after "Name" in the
// column list and populates each item's _columns map with the namespace value.
// This mirrors kubectl's behavior when --all-namespaces is used.
func injectNamespaceColumn(columnNames []string, items []map[string]interface{}) ([]string, []map[string]interface{}) {
	for _, c := range columnNames {
		if strings.EqualFold(c, "Namespace") {
			return columnNames, items
		}
	}

	insertIdx := 0
	for i, c := range columnNames {
		if strings.EqualFold(c, "Name") {
			insertIdx = i + 1
			break
		}
	}

	newCols := make([]string, 0, len(columnNames)+1)
	newCols = append(newCols, columnNames[:insertIdx]...)
	newCols = append(newCols, "Namespace")
	newCols = append(newCols, columnNames[insertIdx:]...)

	for _, item := range items {
		ns, _ := item["namespace"].(string)
		if cells, ok := item[cellsKey].(map[string]interface{}); ok {
			cells["Namespace"] = ns
		}
	}

	return newCols, items
}

func applyLimit(items []map[string]interface{}, limit int) []map[string]interface{} {
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

// renderFilteredTable projects filtered full-struct items back to server-side
// column values and renders as a table.
func renderFilteredTable(items []map[string]interface{}, columnNames []string, opts ptable.Options) string {
	rows := make([][]string, len(items))
	for i, item := range items {
		row := make([]string, len(columnNames))
		cells, _ := item[cellsKey].(map[string]interface{})
		for j, col := range columnNames {
			if cells != nil {
				if v, ok := cells[col]; ok {
					row[j] = fmt.Sprintf("%v", v)
				}
			}
		}
		rows[i] = row
	}

	return ptable.RenderTable("", columnNames, rows, opts)
}

// formatJSONItems marshals items as JSON. Without SELECT, the full Kubernetes
// object is output. With SELECT, only the selected fields are projected.
func formatJSONItems(items []map[string]interface{}, queryOpts *query.QueryOptions) (string, error) {
	if len(items) == 0 {
		return JSONEmpty, nil
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
		b, err := json.MarshalIndent(projected, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshaling JSON: %w", err)
		}
		return string(b), nil
	}

	cleaned := make([]map[string]interface{}, len(items))
	for i, item := range items {
		cleaned[i] = stripInternalKeys(item)
	}
	b, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling JSON: %w", err)
	}
	return string(b), nil
}

// formatYAMLItems marshals items as YAML. Without SELECT, the full Kubernetes
// object is output. With SELECT, only the selected fields are projected.
func formatYAMLItems(items []map[string]interface{}, queryOpts *query.QueryOptions) (string, error) {
	var sb strings.Builder

	if queryOpts.HasSelect {
		for i, item := range items {
			if i > 0 {
				sb.WriteString("---\n")
			}
			p := make(map[string]interface{}, len(queryOpts.Select))
			for _, sel := range queryOpts.Select {
				if v, err := query.GetValue(item, sel.Alias, queryOpts.Select); err == nil && v != nil {
					p[sel.Alias] = v
				}
			}
			b, err := sigyaml.Marshal(p)
			if err != nil {
				return "", fmt.Errorf("marshaling YAML: %w", err)
			}
			sb.Write(b)
		}
		return sb.String(), nil
	}

	for i, item := range items {
		if i > 0 {
			sb.WriteString("---\n")
		}
		b, err := sigyaml.Marshal(stripInternalKeys(item))
		if err != nil {
			return "", fmt.Errorf("marshaling YAML: %w", err)
		}
		sb.Write(b)
	}
	return sb.String(), nil
}

// IsJSONFormat returns true when the output format is "json".
func IsJSONFormat(format string) bool {
	return strings.EqualFold(strings.TrimSpace(format), "json")
}

// JSONError formats an error as a JSON object: {"error": "..."}.
func JSONError(err error) string {
	obj := map[string]string{"error": err.Error()}
	b, _ := json.MarshalIndent(obj, "", "  ")
	return string(b)
}

// JSONEmpty is the canonical empty-array JSON literal.
const JSONEmpty = "[]"

// FlagStr extracts a string value from a flags map.
func FlagStr(flags map[string]any, key string) string {
	v, ok := flags[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	if s != "" {
		return s
	}
	if v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// FlagBool extracts a boolean value from a flags map.
func FlagBool(flags map[string]any, key string) bool {
	v, ok := flags[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return strings.EqualFold(b, "true") || b == "1"
	case float64:
		return b != 0
	default:
		return false
	}
}

// FlagInt extracts an integer value from a flags map.
func FlagInt(flags map[string]any, key string) int {
	v, ok := flags[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		var i int
		_, _ = fmt.Sscanf(n, "%d", &i)
		return i
	default:
		return 0
	}
}
