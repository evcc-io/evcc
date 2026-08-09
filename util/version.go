package util

import (
	"fmt"
	"runtime"
)

const DevVersion = "0.0.0"

var (
	// Version of executable
	Version = DevVersion

	// Commit of executable
	Commit = ""
)

func FormattedVersion() string {
	if Commit != "" {
		return fmt.Sprintf("%s (%s)", Version, Commit)
	}
	return Version
}

// System returns the operating system and architecture
func System() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
