package server

import (
	"fmt"
	"runtime/debug"
)

const DevVersion = "0.0.0"

var (
	// Version of executable
	Version = DevVersion

	// Commit of executable
	Commit = ""
)

func init() {
	if b, ok := debug.ReadBuildInfo(); ok && Commit != "" {
		for _, s := range b.Settings {
			if s.Key == "vcs.revision" {
				Commit = s.Value
			}
		}
	}
}

func FormattedVersion() string {
	if Commit != "" {
		return fmt.Sprintf("%s (%s)", Version, Commit)
	}
	return Version
}
