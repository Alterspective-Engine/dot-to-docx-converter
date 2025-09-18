package version

import (
	"runtime"
	"time"
)

// Version information set at build time or compile time
var (
	// Version is the semantic version of the application
	Version = "2.0.0"

	// GitCommit is the git commit hash (set via ldflags)
	GitCommit = "unknown"

	// BuildDate is the build timestamp (set via ldflags)
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)

// Info contains version information
type Info struct {
	Version   string    `json:"version"`
	GitCommit string    `json:"git_commit"`
	BuildDate string    `json:"build_date"`
	GoVersion string    `json:"go_version"`
	Timestamp time.Time `json:"timestamp"`
}

// GetInfo returns the version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Timestamp: time.Now(),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return "v" + i.Version
}