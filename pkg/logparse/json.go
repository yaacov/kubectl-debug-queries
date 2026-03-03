package logparse

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// k8s timestamp prefix: "2026-03-03T11:13:16.103258979Z " before the JSON object
func stripK8sTimestampPrefix(line string) string {
	if len(line) < 32 {
		return line
	}
	// Look for RFC3339Nano timestamp followed by space and '{'
	idx := strings.Index(line, " {")
	if idx > 0 && idx < 40 && strings.Contains(line[:idx], "T") && strings.HasSuffix(line[:idx], "Z") {
		return line[idx+1:]
	}
	return line
}

// zapKnownKeys are the keys consumed by the renderer prefix and should not appear in Fields.
var zapKnownKeys = map[string]bool{
	"level": true, "ts": true, "msg": true, "logger": true,
	"caller": true, "stacktrace": true,
}

// logrusKnownKeys are the keys consumed by the renderer prefix for logrus-style JSON.
var logrusKnownKeys = map[string]bool{
	"level": true, "timestamp": true, "msg": true, "component": true,
	"pos": true, "time": true,
}

// parseJSONLine parses a single JSON log line into a LogEntry.
func parseJSONLine(line string) LogEntry {
	entry := LogEntry{RawLine: line, Format: FormatJSON}

	stripped := stripK8sTimestampPrefix(strings.TrimSpace(line))
	if len(stripped) == 0 || stripped[0] != '{' {
		return entry
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(stripped), &m); err != nil {
		return entry
	}

	entry.Parsed = true

	if isLogrusStyle(m) {
		parseLogrusFields(m, &entry)
	} else {
		parseZapFields(m, &entry)
	}

	return entry
}

func isLogrusStyle(m map[string]interface{}) bool {
	_, hasComponent := m["component"]
	_, hasPos := m["pos"]
	_, hasTimestamp := m["timestamp"]
	return hasComponent && (hasPos || hasTimestamp)
}

func parseZapFields(m map[string]interface{}, entry *LogEntry) {
	entry.Level = normalizeLevel(strVal(m, "level"))
	entry.Message = strVal(m, "msg")
	entry.Logger = strVal(m, "logger")
	entry.Timestamp = extractTime(strVal(m, "ts"))

	if caller := strVal(m, "caller"); caller != "" {
		entry.Source = caller
	}

	entry.Fields = make(map[string]string)
	for k, v := range m {
		if zapKnownKeys[k] {
			continue
		}
		entry.Fields[k] = flattenValue(v)
	}
}

func parseLogrusFields(m map[string]interface{}, entry *LogEntry) {
	entry.Level = normalizeLevel(strVal(m, "level"))
	entry.Message = strVal(m, "msg")
	entry.Logger = strVal(m, "component")

	ts := strVal(m, "timestamp")
	if ts == "" {
		ts = strVal(m, "time")
	}
	entry.Timestamp = extractTime(ts)

	if pos := strVal(m, "pos"); pos != "" {
		entry.Source = pos
	}

	entry.Fields = make(map[string]string)
	for k, v := range m {
		if logrusKnownKeys[k] {
			continue
		}
		entry.Fields[k] = flattenValue(v)
	}
}

func strVal(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// extractTime pulls HH:MM:SS from various timestamp formats.
func extractTime(ts string) string {
	if ts == "" {
		return ""
	}

	// "2026-03-03 11:13:26.058" or "2026-03-03T11:13:26Z" or "2026-03-03T11:13:26.058Z"
	ts = strings.TrimSuffix(ts, "Z")

	// Find the time portion after 'T' or ' '
	for _, sep := range []string{"T", " "} {
		if idx := strings.Index(ts, sep); idx >= 0 {
			timePart := ts[idx+1:]
			if dot := strings.Index(timePart, "."); dot >= 0 {
				timePart = timePart[:dot]
			}
			if len(timePart) >= 8 {
				return timePart[:8]
			}
			return timePart
		}
	}

	return ts
}

func normalizeLevel(level string) string {
	switch strings.ToLower(level) {
	case "info", "information":
		return "INFO"
	case "warn", "warning":
		return "WARN"
	case "error", "err":
		return "ERROR"
	case "debug", "dbg", "trace":
		return "DEBUG"
	case "fatal", "panic", "critical", "dpanic":
		return "FATAL"
	default:
		return strings.ToUpper(level)
	}
}

// flattenValue converts a value to a compact string for key=val rendering.
func flattenValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]interface{}:
		return flattenMap(val)
	case []interface{}:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = flattenValue(item)
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func flattenMap(m map[string]interface{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(m))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, flattenValue(m[k])))
	}
	return strings.Join(parts, " ")
}

// isJSONLine returns true if the line looks like a JSON log line.
func isJSONLine(line string) bool {
	trimmed := strings.TrimSpace(stripK8sTimestampPrefix(line))
	return len(trimmed) > 0 && trimmed[0] == '{'
}
