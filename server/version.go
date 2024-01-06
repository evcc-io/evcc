package server

import "fmt"

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
