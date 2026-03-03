package logparse

import "testing"

func buildEntries(specs []LogEntry) []LogEntry {
	out := make([]LogEntry, len(specs))
	copy(out, specs)
	return out
}

func TestEnrichInitStageBeforeFirstMarker(t *testing.T) {
	entries := buildEntries([]LogEntry{
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Building command: virt-v2v [...]"},
		{Parsed: true, Logger: "v2v-monitor", Level: "INFO", Message: "Setting up prometheus endpoint :2112/metrics"},
		{Parsed: true, Logger: "libguestfs", Level: "DEBUG", Message: "set_verbose true"},
	})

	enrichVirtV2V(entries)

	for i, e := range entries {
		if e.Fields["stage"] != "init" {
			t.Errorf("entry %d: stage = %q, want %q", i, e.Fields["stage"], "init")
		}
		if e.Fields["progress_pct"] != "" {
			t.Errorf("entry %d: progress_pct = %q, want empty", i, e.Fields["progress_pct"])
		}
	}
}

func TestEnrichPhaseMarkerSetsStage(t *testing.T) {
	entries := buildEntries([]LogEntry{
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Setting up the source: -i libvirt"},
		{Parsed: true, Logger: "libguestfs", Level: "DEBUG", Message: "set_verbose true"},
		{Parsed: true, Logger: "nbdkit", Level: "DEBUG", Message: "transport mode: nbdssl"},
	})

	enrichVirtV2V(entries)

	for i, e := range entries {
		if e.Fields["stage"] != "source-setup" {
			t.Errorf("entry %d: stage = %q, want %q", i, e.Fields["stage"], "source-setup")
		}
	}
}

func TestEnrichStageTransitions(t *testing.T) {
	entries := buildEntries([]LogEntry{
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Setting up the source: -i libvirt"},
		{Parsed: true, Logger: "libguestfs", Level: "DEBUG", Message: "trace call"},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Opening the source"},
		{Parsed: true, Logger: "nbdkit", Level: "DEBUG", Message: "nbdkit 1.38.5"},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Inspecting the source"},
		{Parsed: true, Logger: "guestfsd", Level: "DEBUG", Message: "aug_get"},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Copying disk 1/1"},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Creating output metadata"},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Finishing off"},
	})

	enrichVirtV2V(entries)

	expected := []string{
		"source-setup", "source-setup",
		"source-open", "source-open",
		"inspect", "inspect",
		"disk-copy",
		"metadata",
		"finish",
	}

	for i, want := range expected {
		got := entries[i].Fields["stage"]
		if got != want {
			t.Errorf("entry %d: stage = %q, want %q", i, got, want)
		}
	}
}

func TestEnrichProgressPropagation(t *testing.T) {
	entries := buildEntries([]LogEntry{
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Copying disk 1/1"},
		{Parsed: true, Logger: "v2v-monitor", Level: "INFO", Message: "Progress update, completed 42 %",
			Fields: map[string]string{"progress_pct": "42"}},
		{Parsed: true, Logger: "nbdkit", Level: "DEBUG", Message: "read block"},
		{Parsed: true, Logger: "v2v-monitor", Level: "INFO", Message: "Progress update, completed 85 %",
			Fields: map[string]string{"progress_pct": "85"}},
		{Parsed: true, Logger: "nbdkit", Level: "DEBUG", Message: "write block"},
	})

	enrichVirtV2V(entries)

	expected := []struct {
		stage    string
		progress string
	}{
		{"disk-copy", ""},
		{"disk-copy", "42"},
		{"disk-copy", "42"},
		{"disk-copy", "85"},
		{"disk-copy", "85"},
	}

	for i, want := range expected {
		gotStage := entries[i].Fields["stage"]
		gotPct := entries[i].Fields["progress_pct"]
		if gotStage != want.stage {
			t.Errorf("entry %d: stage = %q, want %q", i, gotStage, want.stage)
		}
		if gotPct != want.progress {
			t.Errorf("entry %d: progress_pct = %q, want %q", i, gotPct, want.progress)
		}
	}
}

func TestEnrichNoProgressOutsideDiskCopy(t *testing.T) {
	entries := buildEntries([]LogEntry{
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Setting up the source: -i libvirt"},
		{Parsed: true, Logger: "libguestfs", Level: "DEBUG", Message: "trace call"},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Opening the source"},
	})

	enrichVirtV2V(entries)

	for i, e := range entries {
		if e.Fields["progress_pct"] != "" {
			t.Errorf("entry %d: progress_pct = %q, want empty (no monitoring lines)", i, e.Fields["progress_pct"])
		}
	}
}

func TestEnrichMultipleConversionRuns(t *testing.T) {
	entries := buildEntries([]LogEntry{
		// Run 1
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Setting up the source: -i libvirt"},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Copying disk 1/1"},
		{Parsed: true, Logger: "v2v-monitor", Level: "INFO", Message: "Progress update, completed 100 %",
			Fields: map[string]string{"progress_pct": "100"}},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Finishing off"},
		// Run 2 -- progress should reset
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Setting up the source: -i disk /var/tmp/v2v/sda"},
		{Parsed: true, Logger: "libguestfs", Level: "DEBUG", Message: "trace call"},
		{Parsed: true, Logger: "v2v", Level: "INFO", Message: "Finishing off"},
	})

	enrichVirtV2V(entries)

	// Run 1: progress propagates
	if entries[3].Fields["progress_pct"] != "100" {
		t.Errorf("run1 finish: progress_pct = %q, want %q", entries[3].Fields["progress_pct"], "100")
	}

	// Run 2: progress resets on source-setup
	if entries[4].Fields["progress_pct"] != "" {
		t.Errorf("run2 source-setup: progress_pct = %q, want empty (should reset)", entries[4].Fields["progress_pct"])
	}
	if entries[5].Fields["progress_pct"] != "" {
		t.Errorf("run2 libguestfs: progress_pct = %q, want empty (should reset)", entries[5].Fields["progress_pct"])
	}
	if entries[6].Fields["stage"] != "finish" {
		t.Errorf("run2 finish: stage = %q, want %q", entries[6].Fields["stage"], "finish")
	}
}

func TestEnrichAllStageKeywords(t *testing.T) {
	tests := []struct {
		message string
		stage   string
	}{
		{"Setting up the source: -i libvirt", "source-setup"},
		{"Opening the source", "source-open"},
		{"Inspecting the source", "inspect"},
		{"Mapping filesystem data to avoid copying unused and blank areas", "map-fs"},
		{"Creating an overlay to protect the source", "overlay"},
		{"Setting up the destination: -o kubevirt", "dest-setup"},
		{"Copying disk 1/1", "disk-copy"},
		{"Creating output metadata", "metadata"},
		{"Finishing off", "finish"},
	}

	for _, tt := range tests {
		t.Run(tt.stage, func(t *testing.T) {
			got := matchStage(tt.message)
			if got != tt.stage {
				t.Errorf("matchStage(%q) = %q, want %q", tt.message, got, tt.stage)
			}
		})
	}
}

func TestEnrichNilFieldsInitialized(t *testing.T) {
	entries := buildEntries([]LogEntry{
		{Parsed: false, RawLine: "some random line"},
	})

	enrichVirtV2V(entries)

	if entries[0].Fields == nil {
		t.Fatal("Fields should be initialized, got nil")
	}
	if entries[0].Fields["stage"] != "init" {
		t.Errorf("stage = %q, want %q", entries[0].Fields["stage"], "init")
	}
}

func TestEnrichEndToEndSmartFormat(t *testing.T) {
	raw := `Building command: virt-v2v [-v -x -o kubevirt]
virt-v2v monitoring: Prometheus progress counter registered.
[   0.0] Setting up the source: -i libvirt
libguestfs: trace: set_verbose true
[ 147.3] Copying disk 1/1
virt-v2v monitoring: Progress update, completed 50 %
nbdkit: debug: read block
[ 358.4] Finishing off`

	entries, det := ParseLines(raw)
	if det.Format != FormatVirtV2V {
		t.Fatalf("expected FormatVirtV2V, got %s", det.Format)
	}

	expectations := []struct {
		stage    string
		progress string
	}{
		{"init", ""},
		{"init", ""},
		{"source-setup", ""},
		{"source-setup", ""},
		{"disk-copy", ""},
		{"disk-copy", "50"},
		{"disk-copy", "50"},
		{"finish", "50"},
	}

	for i, want := range expectations {
		if i >= len(entries) {
			t.Fatalf("entry %d: out of range (only %d entries)", i, len(entries))
		}
		gotStage := entries[i].Fields["stage"]
		gotPct := entries[i].Fields["progress_pct"]
		if gotStage != want.stage {
			t.Errorf("entry %d: stage = %q, want %q", i, gotStage, want.stage)
		}
		if gotPct != want.progress {
			t.Errorf("entry %d: progress_pct = %q, want %q", i, gotPct, want.progress)
		}
	}
}
