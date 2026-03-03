// Package logparse detects and parses structured log formats (JSON, klog, logfmt, CLF)
// and renders them in a compact, uniform format.
package logparse

// Format identifies a log line format.
type Format int

const (
	FormatUnknown Format = iota
	FormatJSON
	FormatKlog
	FormatLogfmt
	FormatCLF
	FormatVirtV2V
	FormatPlain
)

// String returns the human-readable name of the format.
func (f Format) String() string {
	switch f {
	case FormatJSON:
		return "json"
	case FormatKlog:
		return "klog"
	case FormatLogfmt:
		return "logfmt"
	case FormatCLF:
		return "clf"
	case FormatVirtV2V:
		return "virtv2v"
	case FormatPlain:
		return "plain"
	default:
		return "unknown"
	}
}

// LogEntry is the normalized representation of a parsed log line.
type LogEntry struct {
	Timestamp string            // HH:MM:SS (time-of-day only)
	Level     string            // INFO, WARN, ERROR, DEBUG, FATAL
	Message   string            // main log message
	Source    string            // file:line or caller info
	Logger    string            // logger name / component
	Fields    map[string]string // extra context key=value pairs
	RawLine   string            // original unparsed line
	Format    Format            // detected format for this line
	Parsed    bool              // true if the line was successfully parsed
}
