package updater

import (
	"time"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	latest "github.com/tcnksm/go-latest"
)

var instance *updater

type updater struct {
	log   *util.Logger
	cache chan<- util.Param
}

// Run regularly checks version
func Run(log *util.Logger, cache chan<- util.Param) {
	instance = &updater{
		log:   log,
		cache: cache,
	}

	instance.checkVersion()
	for range time.NewTicker(24 * time.Hour).C {
		instance.checkVersion()
	}
}

// checkVersion validates if updates are available
func (u *updater) checkVersion() {
	githubTag := &latest.GithubTag{
		Owner:      "andig",
		Repository: "evcc",
	}

	if res, err := latest.Check(githubTag, server.Version); err == nil {
		if res.Outdated {
			u.log.INFO.Printf("new version available - please upgrade to %s", res.Current)
		}
		u.cache <- util.Param{
			Key: "availableVersion",
			Val: res.Current,
		}
	} else {
		u.log.ERROR.Println("version check failed:", err)
	}
}
