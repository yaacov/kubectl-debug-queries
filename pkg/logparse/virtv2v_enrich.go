package logparse

import "strings"

// stageMapping maps a keyword found in a v2v phase-marker message to a stage name.
var stageMapping = []struct {
	keyword string
	stage   string
}{
	{"Setting up the source", "source-setup"},
	{"Opening the source", "source-open"},
	{"Inspecting the source", "inspect"},
	{"Mapping filesystem", "map-fs"},
	{"Creating an overlay", "overlay"},
	{"Setting up the destination", "dest-setup"},
	{"Copying disk", "disk-copy"},
	{"Creating output metadata", "metadata"},
	{"Finishing off", "finish"},
}

// matchStage returns the stage name if the message contains a known phase keyword.
func matchStage(message string) string {
	for _, m := range stageMapping {
		if strings.Contains(message, m.keyword) {
			return m.stage
		}
	}
	return ""
}

// enrichVirtV2V performs a single forward pass over parsed entries, propagating
// stage and progress_pct fields to every entry based on phase markers and
// monitoring progress lines encountered sequentially.
func enrichVirtV2V(entries []LogEntry) {
	stage := "init"
	progressPct := ""

	for i := range entries {
		e := &entries[i]

		if e.Logger == "v2v" && e.Parsed {
			if s := matchStage(e.Message); s != "" {
				stage = s
				if s == "source-setup" {
					progressPct = ""
				}
			}
		}

		if e.Fields != nil && e.Fields["progress_pct"] != "" {
			progressPct = e.Fields["progress_pct"]
		}

		if e.Fields == nil {
			e.Fields = make(map[string]string)
		}
		e.Fields["stage"] = stage
		if progressPct != "" {
			e.Fields["progress_pct"] = progressPct
		}
	}
}
