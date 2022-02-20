package server

import "fmt"

var (
	// Repository name
	Repository = "evcc"

	// Owner of repository
	Owner = "andig"

	// Version of executable
	Version = "0.0.1-alpha"

	// Commit of executable
	Commit = ""
)

func FormattedVersion() string {
	if Commit != "" {
		return fmt.Sprintf("%s (%s)", Version, Commit)
	}
	return Version
}
