package logparse

import "strings"

const (
	detectSampleSize    = 10
	confidenceThreshold = 0.40
)

// DetectResult holds the detected format and its confidence (0.0-1.0).
type DetectResult struct {
	Format     Format
	Confidence float64
}

// DetectFormat samples the first N non-empty lines to identify the log format.
// Returns FormatPlain with confidence 0 if no format reaches the confidence threshold.
func DetectFormat(text string) DetectResult {
	lines := sampleLines(text, detectSampleSize)
	if len(lines) == 0 {
		return DetectResult{Format: FormatPlain, Confidence: 0}
	}

	votes := map[Format]int{
		FormatVirtV2V: 0,
		FormatJSON:    0,
		FormatKlog:    0,
		FormatLogfmt:  0,
		FormatCLF:     0,
	}

	for _, line := range lines {
		switch {
		case isVirtV2VLine(line):
			votes[FormatVirtV2V]++
		case isJSONLine(line):
			votes[FormatJSON]++
		case isKlogLine(line):
			votes[FormatKlog]++
		case isCLFLine(line):
			votes[FormatCLF]++
		case isLogfmtLine(line):
			votes[FormatLogfmt]++
		}
	}

	// Priority order: VirtV2V first (specialized), then JSON > klog > logfmt > CLF
	priority := []Format{FormatVirtV2V, FormatJSON, FormatKlog, FormatLogfmt, FormatCLF}

	bestFormat := FormatPlain
	bestCount := 0
	for _, f := range priority {
		if votes[f] > bestCount {
			bestCount = votes[f]
			bestFormat = f
		}
	}

	confidence := float64(bestCount) / float64(len(lines))
	if confidence < confidenceThreshold {
		return DetectResult{Format: FormatPlain, Confidence: confidence}
	}

	return DetectResult{Format: bestFormat, Confidence: confidence}
}

// sampleLines returns up to n non-empty lines from text.
func sampleLines(text string, n int) []string {
	var result []string
	for _, line := range strings.SplitN(text, "\n", n*2) {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
			if len(result) >= n {
				break
			}
		}
	}
	return result
}
