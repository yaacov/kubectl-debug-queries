// Package version holds build-time version metadata, injected via ldflags.
package version

var (
	Version   = "0.0.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)
