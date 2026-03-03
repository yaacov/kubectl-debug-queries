package logparse

import (
	"strings"
	"testing"
)

// A realistic sample of the first lines from a virt-v2v conversion pod.
const v2vSample = `Building command: virt-v2v [-v -x -o kubevirt -os /var/tmp/v2v -i libvirt -ic vpx://admin@10.6.46.250/DC/host/10.6.46.29 -- myvm]
Building command: /usr/local/bin/virt-v2v-monitor []
virt-v2v monitoring: Setting up prometheus endpoint :2112/metrics
virt-v2v monitoring: Prometheus progress counter registered.
libguestfs: trace: set_verbose true
libguestfs: trace: set_verbose = 0
info: virt-v2v: virt-v2v 2.7.1rhel=9,release=19.el9 (x86_64)
info: libvirt version: 11.10.0
nbdkit: debug: nbdkit 1.38.5 (nbdkit-1.38.5-12.el9)
nbdkit: vddk[1]: debug: transport mode: nbdssl`

func TestDetectVirtV2V(t *testing.T) {
	det := DetectFormat(v2vSample)
	if det.Format != FormatVirtV2V {
		t.Errorf("expected FormatVirtV2V, got %s", det.Format)
	}
	if det.Confidence < 0.8 {
		t.Errorf("expected confidence >= 0.8, got %f", det.Confidence)
	}
}

func TestDetectNonVirtV2V(t *testing.T) {
	jsonSample := `{"level":"info","ts":"2026-01-01","msg":"hello"}
{"level":"error","ts":"2026-01-01","msg":"world"}`
	det := DetectFormat(jsonSample)
	if det.Format == FormatVirtV2V {
		t.Errorf("JSON log should not be detected as VirtV2V")
	}
}

func TestParsePhaseMarker(t *testing.T) {
	tests := []struct {
		line      string
		timestamp string
		message   string
		diskNum   string
		diskTotal string
	}{
		{
			line:      "[   0.0] Setting up the source: -i libvirt",
			timestamp: "0.0s",
			message:   "Setting up the source: -i libvirt",
		},
		{
			line:      "[ 147.3] Copying disk 1/1",
			timestamp: "147.3s",
			message:   "Copying disk 1/1",
			diskNum:   "1",
			diskTotal: "1",
		},
		{
			line:      "[ 358.4] Finishing off",
			timestamp: "358.4s",
			message:   "Finishing off",
		},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			e := parseVirtV2VLine(tt.line)
			if !e.Parsed {
				t.Fatal("expected line to be parsed")
			}
			if e.Logger != "v2v" {
				t.Errorf("logger: got %q, want %q", e.Logger, "v2v")
			}
			if e.Level != "INFO" {
				t.Errorf("level: got %q, want %q", e.Level, "INFO")
			}
			if e.Timestamp != tt.timestamp {
				t.Errorf("timestamp: got %q, want %q", e.Timestamp, tt.timestamp)
			}
			if e.Message != tt.message {
				t.Errorf("message: got %q, want %q", e.Message, tt.message)
			}
			if tt.diskNum != "" {
				if e.Fields["disk_num"] != tt.diskNum {
					t.Errorf("disk_num: got %q, want %q", e.Fields["disk_num"], tt.diskNum)
				}
				if e.Fields["disk_total"] != tt.diskTotal {
					t.Errorf("disk_total: got %q, want %q", e.Fields["disk_total"], tt.diskTotal)
				}
			}
		})
	}
}

func TestParseMonitorProgress(t *testing.T) {
	e := parseVirtV2VLine("virt-v2v monitoring: Progress update, completed 42 %")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "v2v-monitor" {
		t.Errorf("logger: got %q, want %q", e.Logger, "v2v-monitor")
	}
	if e.Fields["progress_pct"] != "42" {
		t.Errorf("progress_pct: got %q, want %q", e.Fields["progress_pct"], "42")
	}
}

func TestParseMonitorDisk(t *testing.T) {
	e := parseVirtV2VLine("virt-v2v monitoring: Copying disk 1 out of 3")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Fields["disk_num"] != "1" {
		t.Errorf("disk_num: got %q, want %q", e.Fields["disk_num"], "1")
	}
	if e.Fields["disk_total"] != "3" {
		t.Errorf("disk_total: got %q, want %q", e.Fields["disk_total"], "3")
	}
}

func TestParseMonitorFinished(t *testing.T) {
	e := parseVirtV2VLine("virt-v2v monitoring: Finished")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "v2v-monitor" {
		t.Errorf("logger: got %q, want %q", e.Logger, "v2v-monitor")
	}
	if e.Message != "Finished" {
		t.Errorf("message: got %q, want %q", e.Message, "Finished")
	}
}

func TestParseBuildCommand(t *testing.T) {
	e := parseVirtV2VLine("Building command: virt-v2v [-v -x -o kubevirt]")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "v2v" {
		t.Errorf("logger: got %q, want %q", e.Logger, "v2v")
	}
	if !strings.Contains(e.Message, "virt-v2v") {
		t.Errorf("message should contain 'virt-v2v': %q", e.Message)
	}
}

func TestParseLibguestfsTrace(t *testing.T) {
	e := parseVirtV2VLine(`libguestfs: trace: v2v: aug_get "/files/etc/fstab/1/spec"`)
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "libguestfs" {
		t.Errorf("logger: got %q, want %q", e.Logger, "libguestfs")
	}
	if e.Level != "DEBUG" {
		t.Errorf("level: got %q, want %q", e.Level, "DEBUG")
	}
	if !strings.Contains(e.Message, "aug_get") {
		t.Errorf("message should contain 'aug_get': %q", e.Message)
	}
}

func TestParseLibguestfsInfo(t *testing.T) {
	e := parseVirtV2VLine("libguestfs: launch: program=virt-v2v")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "libguestfs" {
		t.Errorf("logger: got %q, want %q", e.Logger, "libguestfs")
	}
	if e.Level != "INFO" {
		t.Errorf("level: got %q, want %q", e.Level, "INFO")
	}
}

func TestParseNbdkit(t *testing.T) {
	tests := []struct {
		line    string
		message string
	}{
		{
			line:    "nbdkit: debug: nbdkit 1.38.5",
			message: "nbdkit 1.38.5",
		},
		{
			line:    "nbdkit: vddk[2]: debug: transport mode: nbdssl",
			message: "transport mode: nbdssl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			e := parseVirtV2VLine(tt.line)
			if !e.Parsed {
				t.Fatal("expected line to be parsed")
			}
			if e.Logger != "nbdkit" {
				t.Errorf("logger: got %q, want %q", e.Logger, "nbdkit")
			}
			if e.Level != "DEBUG" {
				t.Errorf("level: got %q, want %q", e.Level, "DEBUG")
			}
			if e.Message != tt.message {
				t.Errorf("message: got %q, want %q", e.Message, tt.message)
			}
		})
	}
}

func TestParseNbdkitWithEmbeddedTimestamp(t *testing.T) {
	e := parseVirtV2VLine(`nbdkit: vddk[2]: debug: 2026-03-03T00:31:57.225Z warning -[00265] [Originator@6876 sub=transport] SAN mode requires a snapshot.`)
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "nbdkit" {
		t.Errorf("logger: got %q, want %q", e.Logger, "nbdkit")
	}
	if e.Timestamp != "00:31:57" {
		t.Errorf("timestamp: got %q, want %q", e.Timestamp, "00:31:57")
	}
}

func TestParseGuestfsd(t *testing.T) {
	e := parseVirtV2VLine("guestfsd: => aug_get (0x13) took 0.00 secs")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "guestfsd" {
		t.Errorf("logger: got %q, want %q", e.Logger, "guestfsd")
	}
	if e.Level != "DEBUG" {
		t.Errorf("level: got %q, want %q", e.Level, "DEBUG")
	}
}

func TestParseInfoLine(t *testing.T) {
	e := parseVirtV2VLine("info: block device map:")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "v2v-info" {
		t.Errorf("logger: got %q, want %q", e.Logger, "v2v-info")
	}
	if e.Level != "INFO" {
		t.Errorf("level: got %q, want %q", e.Level, "INFO")
	}
}

func TestParseSupermin(t *testing.T) {
	e := parseVirtV2VLine("supermin: build: 185 packages, including dependencies")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "supermin" {
		t.Errorf("logger: got %q, want %q", e.Logger, "supermin")
	}
}

func TestParseDracut(t *testing.T) {
	e := parseVirtV2VLine("dracut: *** Creating image file '/boot/initramfs.img' ***")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "dracut" {
		t.Errorf("logger: got %q, want %q", e.Logger, "dracut")
	}
}

func TestParseKernelDmesg(t *testing.T) {
	e := parseVirtV2VLine("[    0.717989] Booting paravirtualized kernel on KVM")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "kernel" {
		t.Errorf("logger: got %q, want %q", e.Logger, "kernel")
	}
	if e.Level != "DEBUG" {
		t.Errorf("level: got %q, want %q", e.Level, "DEBUG")
	}
	if e.Timestamp != "0.717989s" {
		t.Errorf("timestamp: got %q, want %q", e.Timestamp, "0.717989s")
	}
	if e.Message != "Booting paravirtualized kernel on KVM" {
		t.Errorf("message: got %q", e.Message)
	}
}

func TestPhaseVsKernelDisambiguation(t *testing.T) {
	phase := parseVirtV2VLine("[ 147.3] Copying disk 1/1")
	if phase.Logger != "v2v" {
		t.Errorf("phase marker should have logger 'v2v', got %q", phase.Logger)
	}

	kernel := parseVirtV2VLine("[    0.717989] Booting paravirtualized kernel on KVM")
	if kernel.Logger != "kernel" {
		t.Errorf("kernel line should have logger 'kernel', got %q", kernel.Logger)
	}
}

func TestParseGoHTTPLog(t *testing.T) {
	e := parseVirtV2VLine("2026/03/03 00:38:19 http: superfluous response.WriteHeader call")
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "http" {
		t.Errorf("logger: got %q, want %q", e.Logger, "http")
	}
	if e.Level != "WARN" {
		t.Errorf("level: got %q, want %q", e.Level, "WARN")
	}
	if e.Timestamp != "00:38:19" {
		t.Errorf("timestamp: got %q, want %q", e.Timestamp, "00:38:19")
	}
}

func TestParseAugeasError(t *testing.T) {
	e := parseVirtV2VLine(`augeas failed to parse /etc/profile:`)
	if !e.Parsed {
		t.Fatal("expected line to be parsed")
	}
	if e.Logger != "augeas" {
		t.Errorf("logger: got %q, want %q", e.Logger, "augeas")
	}
	if e.Level != "WARN" {
		t.Errorf("level: got %q, want %q", e.Level, "WARN")
	}
}

func TestParseServerLifecycle(t *testing.T) {
	start := parseVirtV2VLine("Starting server on :8080")
	if !start.Parsed || start.Logger != "server" {
		t.Errorf("start: parsed=%v logger=%q", start.Parsed, start.Logger)
	}

	shutdown := parseVirtV2VLine("Shutdown request received. Shutting down server.")
	if !shutdown.Parsed || shutdown.Logger != "server" {
		t.Errorf("shutdown: parsed=%v logger=%q", shutdown.Parsed, shutdown.Logger)
	}
}

func TestSmartFormatVirtV2V(t *testing.T) {
	output := SmartFormat(v2vSample)
	if !strings.HasPrefix(output, "# format: virtv2v") {
		t.Errorf("SmartFormat header should start with '# format: virtv2v', got: %s",
			strings.SplitN(output, "\n", 2)[0])
	}

	// Verify key rendered components appear.
	if !strings.Contains(output, "[INFO ]") {
		t.Error("output should contain [INFO ] entries")
	}
	if !strings.Contains(output, "v2v:") {
		t.Error("output should contain v2v: logger prefix")
	}
	if !strings.Contains(output, "v2v-monitor:") {
		t.Error("output should contain v2v-monitor: logger prefix")
	}
	if !strings.Contains(output, "nbdkit:") {
		t.Error("output should contain nbdkit: logger prefix")
	}
}

func TestFormatVirtV2VString(t *testing.T) {
	if FormatVirtV2V.String() != "virtv2v" {
		t.Errorf("FormatVirtV2V.String() = %q, want %q", FormatVirtV2V.String(), "virtv2v")
	}
}

func TestParseUnrecognizedLine(t *testing.T) {
	e := parseVirtV2VLine("some random unrecognized output line")
	if e.Parsed {
		t.Error("unrecognized line should not be parsed")
	}
	if e.Format != FormatVirtV2V {
		t.Errorf("format: got %s, want %s", e.Format, FormatVirtV2V)
	}
}
