package server

import (
	"time"

	"github.com/andig/evcc/util"
	latest "github.com/tcnksm/go-latest"
)

var Updater *updater

var (
	// Version of executable
	Version = "0.0.1-alpha"
	// Commit of executable
	Commit = "HEAD"
)

type updater struct {
	log   *util.Logger
	cache chan<- util.Param
}

// RunUpdater regularly checks version
func RunUpdater(log *util.Logger, cache chan<- util.Param) {
	Updater = &updater{
		log:   log,
		cache: cache,
	}

	Updater.checkVersion()
	for range time.NewTicker(24 * time.Hour).C {
		Updater.checkVersion()
	}
}

// checkVersion validates if updates are available
func (u *updater) checkVersion() {
	githubTag := &latest.GithubTag{
		Owner:      "andig",
		Repository: "evcc",
	}

	if res, err := latest.Check(githubTag, Version); err == nil {
		if res.Outdated {
			u.log.INFO.Printf("new version available - please upgrade to %s", res.Current)
		}
		u.cache <- util.Param{
			Key: "availableVersion",
			Val: res.Current,
		}
	} else {
		u.log.ERROR.Printf("version check failed: %v", err)
	}
}
