// Package buildinfo holds version metadata injected at link time with -ldflags -X.
package buildinfo

// These defaults are replaced by the linker when building releases.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
