package logparse

import (
	"fmt"
	"sort"
	"strings"
)

// RenderSmart renders parsed log entries in compact "[LEVEL] HH:MM:SS source: message" format.
// Unparseable lines are prefixed with "[     ]".
func RenderSmart(entries []LogEntry) string {
	var sb strings.Builder
	for i, e := range entries {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if e.Parsed {
			sb.WriteString(renderParsedLine(e))
		} else {
			sb.WriteString("[     ] ")
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

	// extra fields
	if len(e.Fields) > 0 {
		keys := sortedKeys(e.Fields)
		for _, k := range keys {
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
