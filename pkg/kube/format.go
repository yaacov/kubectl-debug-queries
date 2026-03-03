package kube

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaacov/kubectl-debug-queries/pkg/query"
	ptable "github.com/yaacov/kubectl-debug-queries/pkg/table"
)

// FormatTable converts a ServerTable to a rendered string in the given format.
// When queryStr is non-empty, rows are filtered/sorted/limited by the TSL query.
// For JSON output, SELECT fields produce a projected JSON object; for table output,
// the original server-side columns are preserved.
func FormatTable(tbl *ServerTable, format string, opts ptable.Options, queryStr string) (string, error) {
	if format == "" {
		format = "table"
	}

	queryOpts, err := query.ParseQueryString(queryStr)
	if err != nil {
		return "", fmt.Errorf("invalid query: %w", err)
	}

	items := tableToMaps(tbl)

	items, err = query.ApplyQuery(items, queryOpts)
	if err != nil {
		return "", fmt.Errorf("query error: %w", err)
	}

	columnNames := make([]string, len(tbl.Columns))
	for i, c := range tbl.Columns {
		columnNames[i] = c.Name
	}

	switch format {
	case "json":
		return formatJSONItems(items, queryOpts), nil
	case "yaml":
		return formatYAMLItems(items, columnNames, queryOpts), nil
	case "markdown":
		opts.Markdown = true
		return renderFilteredTable(items, columnNames, opts), nil
	default:
		return renderFilteredTable(items, columnNames, opts), nil
	}
}

// tableToMaps converts ServerTable rows to a slice of maps keyed by column name.
func tableToMaps(tbl *ServerTable) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(tbl.Rows))
	for _, r := range tbl.Rows {
		item := make(map[string]interface{}, len(tbl.Columns))
		for j, col := range tbl.Columns {
			if j < len(r.Cells) {
				item[col.Name] = r.Cells[j]
			}
		}
		items = append(items, item)
	}
	return items
}

// renderFilteredTable converts filtered maps back to rows and renders as a table,
// preserving the original server-side columns.
func renderFilteredTable(items []map[string]interface{}, columnNames []string, opts ptable.Options) string {
	rows := make([][]string, len(items))
	for i, item := range items {
		row := make([]string, len(columnNames))
		for j, col := range columnNames {
			if v, ok := item[col]; ok {
				row[j] = fmt.Sprintf("%v", v)
			}
		}
		rows[i] = row
	}

	return ptable.RenderTable("", columnNames, rows, opts)
}

// formatJSONItems marshals items as JSON. When the query has a SELECT clause,
// only the selected fields are included in each object.
func formatJSONItems(items []map[string]interface{}, queryOpts *query.QueryOptions) string {
	if len(items) == 0 {
		return JSONEmpty
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
		b, _ := json.MarshalIndent(projected, "", "  ")
		return string(b)
	}

	b, _ := json.MarshalIndent(items, "", "  ")
	return string(b)
}

// formatYAMLItems formats items as YAML. When the query has a SELECT clause,
// only the selected fields are included.
func formatYAMLItems(items []map[string]interface{}, columnNames []string, queryOpts *query.QueryOptions) string {
	var sb strings.Builder

	if queryOpts.HasSelect {
		for i, item := range items {
			if i > 0 {
				sb.WriteString("---\n")
			}
			for _, sel := range queryOpts.Select {
				if v, err := query.GetValue(item, sel.Alias, queryOpts.Select); err == nil && v != nil {
					sb.WriteString(fmt.Sprintf("%s: %v\n", sel.Alias, v))
				}
			}
		}
		return sb.String()
	}

	for i, item := range items {
		if i > 0 {
			sb.WriteString("---\n")
		}
		for _, col := range columnNames {
			if v, ok := item[col]; ok {
				sb.WriteString(fmt.Sprintf("%s: %v\n", col, v))
			}
		}
	}
	return sb.String()
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
