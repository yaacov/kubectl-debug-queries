package logparse

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseLines detects the log format, then parses all lines into LogEntry structs.
func ParseLines(text string) ([]LogEntry, DetectResult) {
	det := DetectFormat(text)
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")

	entries := make([]LogEntry, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		entries = append(entries, parseLine(line, det.Format))
	}
	return entries, det
}

func parseLine(line string, format Format) LogEntry {
	switch format {
	case FormatJSON:
		return parseJSONLine(line)
	case FormatKlog:
		return parseKlogLine(line)
	case FormatLogfmt:
		return parseLogfmtLine(line)
	case FormatCLF:
		return parseCLFLine(line)
	default:
		return LogEntry{RawLine: line, Format: FormatPlain, Parsed: false}
	}
}

// SmartFormat auto-detects the format, parses, and renders compact output.
// This is the default log output mode.
func SmartFormat(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}

	entries, det := ParseLines(text)
	if det.Format == FormatPlain || len(entries) == 0 {
		return text
	}

	header := fmt.Sprintf("# format: %s, lines: %d", det.Format, len(entries))
	body := RenderSmart(entries)
	return header + "\n" + body
}

// JSONFormat auto-detects, parses, and returns a JSON array of LogEntry objects.
func JSONFormat(text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "[]", nil
	}

	entries, _ := ParseLines(text)

	type jsonEntry struct {
		Timestamp string            `json:"timestamp,omitempty"`
		Level     string            `json:"level,omitempty"`
		Message   string            `json:"message,omitempty"`
		Source    string            `json:"source,omitempty"`
		Logger    string            `json:"logger,omitempty"`
		Fields    map[string]string `json:"fields,omitempty"`
		RawLine   string            `json:"raw_line"`
		Format    string            `json:"format"`
		Parsed    bool              `json:"parsed"`
	}

	out := make([]jsonEntry, len(entries))
	for i, e := range entries {
		out[i] = jsonEntry{
			Timestamp: e.Timestamp,
			Level:     e.Level,
			Message:   e.Message,
			Source:    e.Source,
			Logger:    e.Logger,
			Fields:    e.Fields,
			RawLine:   e.RawLine,
			Format:    e.Format.String(),
			Parsed:    e.Parsed,
		}
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling log entries: %w", err)
	}
	return string(b), nil
}
