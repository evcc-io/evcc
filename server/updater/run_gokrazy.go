//go:build gokrazy

package updater

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/google/go-github/v32/github"
)

var latest *github.RepositoryRelease

// Run regularly checks version
func Run(log *util.Logger, httpd webServer, outChan chan<- util.Param) {
	u := &watch{
		log:     log,
		outChan: outChan,
		repo:    NewRepo(owner, repository),
	}

	httpd.Router().PathPrefix("/api/update").HandlerFunc(u.updateHandler)

	c := make(chan *github.RepositoryRelease, 1)
	go u.watchReleases(server.Version, c) // endless

	// signal update support
	u.Send("hasUpdater", true)

	for rel := range c {
		latest = rel
		u.Send("availableVersion", *latest.TagName)
	}
}

const rootFSAsset = "evcc_%s.rootfs.gz"

func (u *watch) updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || latest == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	name := fmt.Sprintf(rootFSAsset, *latest.TagName)
	assetID, size, err := u.repo.FindReleaseAsset(name)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "rootfs image not found: %v", err)
		return
	}

	if err := u.execute(assetID, size); err != nil {
		u.log.ERROR.Printf("could not find release image: %v", err)

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "update failed: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
