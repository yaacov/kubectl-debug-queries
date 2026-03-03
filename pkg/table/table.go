package table

import (
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// RenderTable renders column headers and string rows as a pretty table.
// Columns and row cells come directly from the K8s server-side table response.
func RenderTable(title string, columns []string, rows [][]string, opts Options) string {
	if len(rows) == 0 {
		return "(no results)"
	}

	if opts.SortBy != "" {
		rows = sortRows(columns, rows, opts.SortBy)
	}

	if opts.Limit > 0 && len(rows) > opts.Limit {
		rows = rows[:opts.Limit]
	}

	t := newTableWriter(title)

	header := make(table.Row, len(columns))
	for i, c := range columns {
		header[i] = strings.ToUpper(c)
	}
	t.AppendHeader(header)

	for _, r := range rows {
		row := make(table.Row, len(r))
		for i, cell := range r {
			row[i] = cell
		}
		t.AppendRow(row)
	}

	return renderTable(t, opts.Markdown)
}

func renderTable(t table.Writer, markdown bool) string {
	if markdown {
		return t.RenderMarkdown()
	}
	return t.Render()
}

func newTableWriter(title string) table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleDefault)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.Style().Options.SeparateHeader = true
	t.Style().Options.SeparateRows = false
	t.Style().Options.SeparateFooter = false
	t.Style().Format.HeaderAlign = text.AlignLeft
	t.Style().Box.PaddingLeft = ""
	t.Style().Box.PaddingRight = "  "
	if title != "" {
		t.SetTitle(title)
	}
	return t
}

// sortRows sorts rows by the column matching sortBy (case-insensitive).
// Falls back to original order if the column is not found.
func sortRows(columns []string, rows [][]string, sortBy string) [][]string {
	idx := -1
	for i, c := range columns {
		if strings.EqualFold(c, sortBy) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return rows
	}

	sorted := make([][]string, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := "", ""
		if idx < len(sorted[i]) {
			a = sorted[i][idx]
		}
		if idx < len(sorted[j]) {
			b = sorted[j][idx]
		}
		return strings.ToLower(a) < strings.ToLower(b)
	})
	return sorted
}
