package logparse

import (
	"regexp"
)

// klogRe matches standard klog format:
// I0303 11:14:07.353271       1 metrics.go:637] message text
var klogRe = regexp.MustCompile(`^([IWEF])(\d{4})\s+(\d{2}:\d{2}:\d{2})\.\d+\s+\d+\s+(\S+:\d+)\]\s+(.*)$`)

var klogLevelMap = map[byte]string{
	'I': "INFO",
	'W': "WARN",
	'E': "ERROR",
	'F': "FATAL",
}

// parseKlogLine parses a standard klog line or its structured variant.
func parseKlogLine(line string) LogEntry {
	entry := LogEntry{RawLine: line, Format: FormatKlog}

	m := klogRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}

	entry.Parsed = true
	entry.Level = klogLevelMap[m[1][0]]
	entry.Timestamp = m[3]
	entry.Source = m[4]

	msg := m[5]
	entry.Fields = make(map[string]string)

	// Structured klog: "msg"="value" "key"="value" ...
	if len(msg) > 0 && msg[0] == '"' {
		parseKlogStructured(msg, &entry)
	} else {
		entry.Message = msg
	}

	return entry
}

// parseKlogStructured extracts "key"="value" pairs from klog structured output.
func parseKlogStructured(msg string, entry *LogEntry) {
	pairs := klogStructuredRe.FindAllStringSubmatch(msg, -1)
	for _, p := range pairs {
		key := p[1]
		val := p[2]
		switch key {
		case "msg":
			entry.Message = val
		case "logger":
			entry.Logger = val
		default:
			entry.Fields[key] = val
		}
	}
	if entry.Message == "" {
		entry.Message = msg
	}
}

// klogStructuredRe matches "key"="value" pairs in klog structured logging output.
var klogStructuredRe = regexp.MustCompile(`"([^"]+)"="([^"]*)"`)

// isKlogLine returns true if the line looks like a klog-formatted line.
func isKlogLine(line string) bool {
	if len(line) < 5 {
		return false
	}
	c := line[0]
	return (c == 'I' || c == 'W' || c == 'E' || c == 'F') &&
		line[1] >= '0' && line[1] <= '9' &&
		line[2] >= '0' && line[2] <= '9' &&
		line[3] >= '0' && line[3] <= '9' &&
		line[4] >= '0' && line[4] <= '9'
}
