package logparse

import (
	"fmt"
	"sort"
	"strings"
)

// stageColumnWidth is the fixed width for the stage column in rendered output.
// Matches the longest stage name ("source-setup" = 12 chars).
const stageColumnWidth = 12

// RenderSmart renders parsed log entries in compact "[LEVEL] stage source: message" format.
// Unparseable lines are prefixed with "[    ]".
func RenderSmart(entries []LogEntry) string {
	var sb strings.Builder
	for i, e := range entries {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if e.Parsed {
			sb.WriteString(renderParsedLine(e))
		} else {
			sb.WriteString("[    ] ")
			if stage := e.Fields["stage"]; stage != "" {
				sb.WriteString(fmt.Sprintf("%-*s ", stageColumnWidth, stage))
			}
			sb.WriteString(e.RawLine)
		}
	}
	return sb.String()
}

func renderParsedLine(e LogEntry) string {
	var sb strings.Builder

	// [LEVEL]
	level := e.Level
	if level == "" {
		level = "INFO"
	}
	sb.WriteString(fmt.Sprintf("[%-5s] ", level))

	// stage (fixed-width column, only when present)
	if stage := e.Fields["stage"]; stage != "" {
		sb.WriteString(fmt.Sprintf("%-*s ", stageColumnWidth, stage))
	}

	// HH:MM:SS
	if e.Timestamp != "" {
		sb.WriteString(e.Timestamp)
		sb.WriteByte(' ')
	}

	// source: (logger or file:line)
	source := e.Logger
	if source == "" {
		source = e.Source
	}
	if source != "" {
		// Strip random suffixes from logger names like "plan|v6z6r" -> "plan"
		if idx := strings.Index(source, "|"); idx > 0 {
			source = source[:idx]
		}
		sb.WriteString(source)
		sb.WriteString(": ")
	}

	// message
	sb.WriteString(e.Message)

	// extra fields (skip "stage" -- already rendered as a column)
	if len(e.Fields) > 0 {
		keys := sortedKeys(e.Fields)
		for _, k := range keys {
			if k == "stage" {
				continue
			}
			v := e.Fields[k]
			if v == "" {
				continue
			}
			sb.WriteByte(' ')
			sb.WriteString(k)
			sb.WriteByte('=')
			if strings.Contains(v, " ") {
				sb.WriteByte('"')
				sb.WriteString(v)
				sb.WriteByte('"')
			} else {
				sb.WriteString(v)
			}
		}
	}

	return sb.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
