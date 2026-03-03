package logparse

import (
	"strings"
)

// parseLogfmtLine parses a logfmt-style "key=value key=value ..." log line.
func parseLogfmtLine(line string) LogEntry {
	entry := LogEntry{RawLine: line, Format: FormatLogfmt}

	pairs := parseLogfmtPairs(line)
	if len(pairs) < 2 {
		return entry
	}

	entry.Parsed = true
	entry.Fields = make(map[string]string)

	for _, kv := range pairs {
		switch kv.key {
		case "time", "ts":
			entry.Timestamp = extractTime(kv.value)
		case "level":
			entry.Level = normalizeLevel(kv.value)
		case "source", "caller":
			entry.Source = kv.value
		case "msg":
			entry.Message = kv.value
		case "logger":
			entry.Logger = kv.value
		default:
			entry.Fields[kv.key] = kv.value
		}
	}

	return entry
}

type kvPair struct {
	key   string
	value string
}

// parseLogfmtPairs extracts key=value pairs from a logfmt line.
// Values can be quoted ("...") or unquoted (until next space).
func parseLogfmtPairs(line string) []kvPair {
	var pairs []kvPair
	s := strings.TrimSpace(line)

	for len(s) > 0 {
		eqIdx := strings.Index(s, "=")
		if eqIdx < 0 {
			break
		}

		key := s[:eqIdx]
		// Key is the last whitespace-delimited token before '='
		if spIdx := strings.LastIndex(key, " "); spIdx >= 0 {
			key = key[spIdx+1:]
		}

		rest := s[eqIdx+1:]
		var value string

		if len(rest) > 0 && rest[0] == '"' {
			// Quoted value
			endQuote := strings.Index(rest[1:], "\"")
			if endQuote >= 0 {
				value = rest[1 : endQuote+1]
				rest = rest[endQuote+2:]
			} else {
				value = rest[1:]
				rest = ""
			}
		} else {
			// Unquoted value: until next space
			spIdx := strings.Index(rest, " ")
			if spIdx >= 0 {
				value = rest[:spIdx]
				rest = rest[spIdx+1:]
			} else {
				value = rest
				rest = ""
			}
		}

		if key != "" {
			pairs = append(pairs, kvPair{key: key, value: value})
		}
		s = rest
	}

	return pairs
}

// isLogfmtLine returns true if the line has 3+ key=value pairs, suggesting logfmt.
func isLogfmtLine(line string) bool {
	pairs := parseLogfmtPairs(line)
	return len(pairs) >= 3
}
