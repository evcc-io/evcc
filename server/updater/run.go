// +build !gokrazy

package updater

import (
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/google/go-github/v32/github"
)

// Run regularly checks version
func Run(log *util.Logger, httpd webServer, tee util.TeeAttacher, outChan chan<- util.Param) {
	u := &watch{
		log:     log,
		outChan: outChan,
		repo:    NewRepo(server.Owner, server.Repository),
	}

	c := make(chan *github.RepositoryRelease, 1)
	go u.watchReleases(server.Version, c) // endless

	for rel := range c {
		u.Send("availableVersion", *rel.TagName)
	}
}
