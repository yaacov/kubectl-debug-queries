// Package table provides pretty-printed table rendering for Kubernetes resource results.
package table

import "time"

// Options controls how Kubernetes results are rendered as tables.
type Options struct {
	// LocalTime renders timestamps in the local timezone instead of UTC.
	LocalTime bool

	// Markdown renders the table in GitHub-compatible Markdown format.
	Markdown bool

	// SortBy is the column name to sort rows by (case-insensitive match).
	SortBy string

	// Limit caps the number of rows rendered. Zero means no limit.
	Limit int
}

const dateFormat = "2006-01-02 15:04:05"

// FormatTimestamp converts a Unix timestamp to a human-readable string.
func (o Options) FormatTimestamp(ts float64) string {
	t := time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
	if !o.LocalTime {
		t = t.UTC()
	}
	return t.Format(dateFormat)
}
