package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of the application
	Version = "dev"
	// GitCommit is the git commit hash
	GitCommit = "none"
	// BuildTime is the build time
	BuildTime = "unknown"
)

// Info returns the version information
func Info() string {
	return fmt.Sprintf("Version: %s, GitCommit: %s, BuildTime: %s, GoVersion: %s, OS/Arch: %s/%s",
		Version, GitCommit, BuildTime, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// Short returns a short version string
func Short() string {
	return fmt.Sprintf("%s-%s", Version, GitCommit)
}
