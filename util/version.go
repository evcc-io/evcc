package util

import "runtime"

const DevVersion = "0.0.0"

var (
	// Version of executable
	Version = DevVersion

	// Commit of executable
	Commit = ""
)

// FormattedVersion returns the version, untagged builds carry the commit as build metadata
func FormattedVersion() string {
	return Version
}

// System returns the operating system and architecture
func System() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
