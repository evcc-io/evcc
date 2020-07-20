package util

import (
	"github.com/andig/evcc/server"
	latest "github.com/tcnksm/go-latest"
)

var Updater *updater

var (
	// Version of executable
	Version = "dev"
	// Commit of executable
	Commit = "HEAD"
)

type Updater struct {
	log *Logger
}

func (u Updater) Run(log *Logger) {
	if updater == nil {
		updater = &Updater{
			log: log,
		}
	}

	go updater.run()
}

// checkVersion validates if updates are available
func (u *Updater) checkVersion() {
	githubTag := &latest.GithubTag{
		Owner:      "andig",
		Repository: "evcc",
	}

	if res, err := latest.Check(githubTag, server.Version); err == nil {
		if res.Outdated {
			u.log.INFO.Printf("new version available - please upgrade to %s", res.Current)
		}
	}
}

// checkVersion validates if updates are available
func (u *Updater) run() {
	u.checkVersion()
}
