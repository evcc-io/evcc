package server

import (
	"github.com/andig/evcc/util"
	latest "github.com/tcnksm/go-latest"
)

var Updater *updater

var (
	// Version of executable
	Version = "dev"
	// Commit of executable
	Commit = "HEAD"
)

type updater struct {
	log *util.Logger
}

func RunUpdater(log *util.Logger) {
	Updater = &updater{
		log: log,
	}

	go Updater.run()
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
	}
}

// checkVersion validates if updates are available
func (u *updater) run() {
	u.checkVersion()
}
