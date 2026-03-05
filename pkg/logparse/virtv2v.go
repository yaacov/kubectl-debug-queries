package logparse

import (
	"regexp"
	"strings"
)

// Regexes for virt-v2v log line classification.
var (
	// Phase markers: [   0.0] Setting up the source ...
	// 1-2 decimal digits distinguish from kernel dmesg (6 digits).
	v2vPhaseRe = regexp.MustCompile(`^\[\s+(\d+\.\d{1,2})\]\s+(.*)`)

	// Kernel dmesg: [    0.717989] Booting paravirtualized kernel on KVM
	v2vKernelRe = regexp.MustCompile(`^\[\s+(\d+\.\d{3,})\]\s+(.*)`)

	// Monitoring: virt-v2v monitoring: Progress update, completed 42 %
	v2vMonitorRe = regexp.MustCompile(`^virt-v2v monitoring:\s+(.*)`)

	// Progress percentage inside monitoring line.
	v2vProgressRe = regexp.MustCompile(`completed\s+(\d+)\s*%`)

	// Disk info: "Copying disk 1 out of 1" or "Copying disk 1/1"
	v2vDiskMonitorRe = regexp.MustCompile(`[Cc]opying disk (\d+)\s+out of\s+(\d+)`)
	v2vDiskPhaseRe   = regexp.MustCompile(`[Cc]opying disk (\d+)/(\d+)`)

	// Building command: virt-v2v [...]
	v2vBuildCmdRe = regexp.MustCompile(`^Building command:\s+(.*)`)

	// libguestfs trace: libguestfs: trace: [module:] <func> <args>
	v2vLibguestfsTraceRe = regexp.MustCompile(`^libguestfs:\s+trace:\s+(?:\w+:\s+)?(.*)`)

	// libguestfs other: libguestfs: <anything>
	v2vLibguestfsRe = regexp.MustCompile(`^libguestfs:\s+(.*)`)

	// nbdkit debug: nbdkit: [vddk[N]:] debug: <message>
	v2vNbdkitRe = regexp.MustCompile(`^nbdkit:\s+(?:vddk\[\d+\]:\s+)?(?:debug:\s+)?(.*)`)

	// Embedded ISO timestamp in nbdkit/VDDK messages: 2026-03-03T00:31:57.225Z
	v2vEmbeddedTsRe = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T(\d{2}:\d{2}:\d{2})\.\d+Z)`)

	// guestfsd: <= or => or error ...
	v2vGuestfsdRe = regexp.MustCompile(`^guestfsd:\s+(.*)`)

	// info: <message>
	v2vInfoRe = regexp.MustCompile(`^info:\s+(.*)`)

	// supermin: <message>
	v2vSuperminRe = regexp.MustCompile(`^supermin:\s+(.*)`)

	// dracut: <message>
	v2vDracutRe = regexp.MustCompile(`^dracut:\s+(.*)`)

	// Go HTTP log: 2026/03/03 00:38:19 http: ...
	v2vGoHTTPRe = regexp.MustCompile(`^\d{4}/\d{2}/\d{2}\s+(\d{2}:\d{2}:\d{2})\s+(.*)`)

	// augeas errors: augeas failed to parse ...
	v2vAugeasRe = regexp.MustCompile(`^augeas\s+failed\s+(.*)`)

	// libnbd debug: libnbd: debug: nbd1: nbd_connect_uri: ...
	v2vLibnbdRe = regexp.MustCompile(`^libnbd:\s+debug:\s+(\w+):\s+(.*)`)

	// check_host_free_space: large_tmpdir=/var/tmp free_space=49187164160
	v2vCheckHostRe = regexp.MustCompile(`^check_host_free_space:\s+(.*)`)

	// XML content lines from libvirt XML dump or inspection output.
	v2vXMLTagRe = regexp.MustCompile(`^\s*<[?/]?\w+`)

	// Cleanup: rm -rf -- '/tmp/...'
	v2vCleanupRe = regexp.MustCompile(`^rm\s+-rf\s+--\s+(.*)`)
)

// isVirtV2VLine returns true if the line looks like a virt-v2v or virt-v2v-inspector pod log line.
func isVirtV2VLine(line string) bool {
	switch {
	case strings.HasPrefix(line, "Building command:"):
		return true
	case strings.HasPrefix(line, "virt-v2v monitoring:"):
		return true
	case strings.HasPrefix(line, "libguestfs:"):
		return true
	case strings.HasPrefix(line, "nbdkit:"):
		return true
	case strings.HasPrefix(line, "libnbd:"):
		return true
	case strings.HasPrefix(line, "guestfsd:"):
		return true
	case strings.HasPrefix(line, "supermin:"):
		return true
	case strings.HasPrefix(line, "dracut:"):
		return true
	case strings.HasPrefix(line, "info: ") && strings.Contains(line, "virt-v2v"):
		return true
	case strings.HasPrefix(line, "augeas "):
		return true
	case strings.HasPrefix(line, "check_host_free_space:"):
		return true
	case strings.HasPrefix(line, "libvirt xml is"):
		return true
	case strings.HasPrefix(line, "running nbdkit:"):
		return true
	case v2vPhaseRe.MatchString(line):
		return true
	default:
		return false
	}
}

// parseVirtV2VLine parses a single virt-v2v or virt-v2v-inspector pod log line.
func parseVirtV2VLine(line string) LogEntry {
	entry := LogEntry{RawLine: line, Format: FormatVirtV2V}

	switch {
	case strings.HasPrefix(line, "Building command:"):
		return parseV2VBuildCmd(line, entry)

	case strings.HasPrefix(line, "virt-v2v monitoring:"):
		return parseV2VMonitor(line, entry)

	case strings.HasPrefix(line, "libguestfs:"):
		return parseV2VLibguestfs(line, entry)

	case strings.HasPrefix(line, "nbdkit:"):
		return parseV2VNbdkit(line, entry)

	case strings.HasPrefix(line, "libnbd:"):
		return parseV2VLibnbd(line, entry)

	case strings.HasPrefix(line, "guestfsd:"):
		return parseV2VGuestfsd(line, entry)

	case strings.HasPrefix(line, "info:"):
		return parseV2VInfo(line, entry)

	case strings.HasPrefix(line, "supermin:"):
		return parseV2VSupermin(line, entry)

	case strings.HasPrefix(line, "dracut:"):
		return parseV2VDracut(line, entry)

	case strings.HasPrefix(line, "augeas "):
		return parseV2VAugeas(line, entry)

	case strings.HasPrefix(line, "check_host_free_space:"):
		return parseV2VCheckHost(line, entry)

	case strings.HasPrefix(line, "libvirt xml is"):
		entry.Parsed = true
		entry.Logger = "v2v"
		entry.Level = "INFO"
		entry.Message = line
		return entry

	case strings.HasPrefix(line, "running nbdkit:"):
		entry.Parsed = true
		entry.Logger = "v2v"
		entry.Level = "INFO"
		entry.Message = line
		return entry

	case strings.HasPrefix(line, "rm -rf "):
		return parseV2VCleanup(line, entry)

	case strings.HasPrefix(line, "Starting server"):
		entry.Parsed = true
		entry.Logger = "server"
		entry.Level = "INFO"
		entry.Message = line
		return entry

	case strings.HasPrefix(line, "Shutdown"):
		entry.Parsed = true
		entry.Logger = "server"
		entry.Level = "INFO"
		entry.Message = line
		return entry
	}

	// Go HTTP log: 2026/03/03 00:38:19 http: ...
	if m := v2vGoHTTPRe.FindStringSubmatch(line); m != nil {
		entry.Parsed = true
		entry.Logger = "http"
		entry.Level = "WARN"
		entry.Timestamp = m[1]
		entry.Message = m[2]
		return entry
	}

	// Phase marker vs kernel dmesg: phase markers have 1-2 decimal digits,
	// kernel has 3+ decimal digits. Check phase first (more specific).
	if m := v2vPhaseRe.FindStringSubmatch(line); m != nil {
		// Verify it's not actually a kernel line (3+ decimals).
		if !v2vKernelRe.MatchString(line) {
			return parseV2VPhase(m, entry)
		}
	}

	// Kernel dmesg -- preserve boot-relative timestamp.
	if m := v2vKernelRe.FindStringSubmatch(line); m != nil {
		entry.Parsed = true
		entry.Logger = "kernel"
		entry.Level = "DEBUG"
		entry.Timestamp = m[1] + "s"
		entry.Message = m[2]
		return entry
	}

	// XML content lines (libvirt XML dump or v2v-inspection output).
	if v2vXMLTagRe.MatchString(line) {
		entry.Parsed = true
		entry.Logger = "xml"
		entry.Level = "INFO"
		entry.Message = line
		return entry
	}

	return entry
}

func parseV2VPhase(m []string, entry LogEntry) LogEntry {
	entry.Parsed = true
	entry.Logger = "v2v"
	entry.Level = "INFO"
	entry.Timestamp = m[1] + "s"
	entry.Message = m[2]
	entry.Fields = make(map[string]string)

	if dm := v2vDiskPhaseRe.FindStringSubmatch(m[2]); dm != nil {
		entry.Fields["disk_num"] = dm[1]
		entry.Fields["disk_total"] = dm[2]
	}

	return entry
}

func parseV2VMonitor(line string, entry LogEntry) LogEntry {
	m := v2vMonitorRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "v2v-monitor"
	entry.Level = "INFO"
	entry.Message = m[1]
	entry.Fields = make(map[string]string)

	if pm := v2vProgressRe.FindStringSubmatch(m[1]); pm != nil {
		entry.Fields["progress_pct"] = pm[1]
	}
	if dm := v2vDiskMonitorRe.FindStringSubmatch(m[1]); dm != nil {
		entry.Fields["disk_num"] = dm[1]
		entry.Fields["disk_total"] = dm[2]
	}

	return entry
}

func parseV2VBuildCmd(line string, entry LogEntry) LogEntry {
	m := v2vBuildCmdRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "v2v"
	entry.Level = "INFO"
	entry.Message = "Building command: " + m[1]
	return entry
}

func parseV2VLibguestfs(line string, entry LogEntry) LogEntry {
	entry.Logger = "libguestfs"

	if m := v2vLibguestfsTraceRe.FindStringSubmatch(line); m != nil {
		entry.Parsed = true
		entry.Level = "DEBUG"
		entry.Message = m[1]
		return entry
	}

	if m := v2vLibguestfsRe.FindStringSubmatch(line); m != nil {
		entry.Parsed = true
		entry.Level = "INFO"
		entry.Message = m[1]
		return entry
	}

	return entry
}

func parseV2VNbdkit(line string, entry LogEntry) LogEntry {
	m := v2vNbdkitRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "nbdkit"
	entry.Level = "DEBUG"
	entry.Message = m[1]

	if ts := v2vEmbeddedTsRe.FindStringSubmatch(m[1]); ts != nil {
		entry.Timestamp = ts[2]
	}

	return entry
}

func parseV2VGuestfsd(line string, entry LogEntry) LogEntry {
	m := v2vGuestfsdRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "guestfsd"
	entry.Level = "DEBUG"
	entry.Message = m[1]
	return entry
}

func parseV2VInfo(line string, entry LogEntry) LogEntry {
	m := v2vInfoRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "v2v-info"
	entry.Level = "INFO"
	entry.Message = m[1]
	return entry
}

func parseV2VSupermin(line string, entry LogEntry) LogEntry {
	m := v2vSuperminRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "supermin"
	entry.Level = "INFO"
	entry.Message = m[1]
	return entry
}

func parseV2VDracut(line string, entry LogEntry) LogEntry {
	m := v2vDracutRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "dracut"
	entry.Level = "INFO"
	entry.Message = m[1]
	return entry
}

func parseV2VAugeas(line string, entry LogEntry) LogEntry {
	m := v2vAugeasRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "augeas"
	entry.Level = "WARN"
	entry.Message = "failed " + m[1]
	return entry
}

func parseV2VLibnbd(line string, entry LogEntry) LogEntry {
	m := v2vLibnbdRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "libnbd"
	entry.Level = "DEBUG"
	entry.Message = m[1] + ": " + m[2]
	return entry
}

func parseV2VCheckHost(line string, entry LogEntry) LogEntry {
	m := v2vCheckHostRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "v2v"
	entry.Level = "INFO"
	entry.Message = "check_host_free_space: " + m[1]
	return entry
}

func parseV2VCleanup(line string, entry LogEntry) LogEntry {
	m := v2vCleanupRe.FindStringSubmatch(line)
	if m == nil {
		return entry
	}
	entry.Parsed = true
	entry.Logger = "v2v"
	entry.Level = "INFO"
	entry.Message = "cleanup: rm -rf " + m[1]
	return entry
}
