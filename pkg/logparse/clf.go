package logparse

import (
	"regexp"
	"strings"
	"time"
)

// clfRe matches Common Log Format:
// ::ffff:10.128.0.2 - - [03/Mar/2026 11:14:56] "GET / HTTP/1.1" 200 -
// 192.168.1.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326
var clfRe = regexp.MustCompile(`^([\S:]+)\s+(\S+)\s+(\S+)\s+\[([^\]]+)\]\s+"([^"]+)"\s+(\d+)\s+(\S+)`)

// parseCLFLine parses a Common/Combined Log Format line.
func parseCLFLine(line string) LogEntry {
	entry := LogEntry{RawLine: line, Format: FormatCLF}

	m := clfRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}

	entry.Parsed = true
	entry.Timestamp = extractCLFTime(m[4])

	status := m[6]
	entry.Level = clfStatusToLevel(status)

	entry.Message = m[5]
	entry.Fields = map[string]string{
		"client": m[1],
		"status": status,
		"size":   m[7],
	}
	if m[3] != "-" {
		entry.Fields["user"] = m[3]
	}

	return entry
}

// extractCLFTime pulls HH:MM:SS from CLF timestamp like "03/Mar/2026 11:14:56"
// or "10/Oct/2000:13:55:36 -0700".
func extractCLFTime(ts string) string {
	// Try "DD/Mon/YYYY HH:MM:SS" (Python-style)
	for _, layout := range []string{
		"02/Jan/2006 15:04:05",
		"02/Jan/2006:15:04:05 -0700",
	} {
		if t, err := time.Parse(layout, ts); err == nil {
			return t.Format("15:04:05")
		}
	}

	// Fallback: find HH:MM:SS anywhere in the string
	for i := 0; i+8 <= len(ts); i++ {
		seg := ts[i : i+8]
		if len(seg) == 8 && seg[2] == ':' && seg[5] == ':' {
			return seg
		}
	}

	return ""
}

func clfStatusToLevel(status string) string {
	if len(status) == 0 {
		return "INFO"
	}
	switch status[0] {
	case '5':
		return "ERROR"
	case '4':
		return "WARN"
	default:
		return "INFO"
	}
}

// isCLFLine returns true if the line matches Common Log Format.
func isCLFLine(line string) bool {
	return clfRe.MatchString(strings.TrimSpace(line))
}
