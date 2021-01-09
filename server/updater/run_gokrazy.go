// +build gokrazy

package updater

import (
	"fmt"
	"net/http"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/google/go-github/v32/github"
)

var (
	latest *github.RepositoryRelease
)

// Run regularly checks version
func Run(log *util.Logger, httpd webServer, tee util.TeeAttacher, outChan chan<- util.Param) {
	u := &watch{
		log:     log,
		outChan: outChan,
		repo:    NewRepo(server.Owner, server.Repository),
	}

	httpd.Router().PathPrefix("/api/update").HandlerFunc(u.updateHandler)

	c := make(chan *github.RepositoryRelease, 1)
	go u.watchReleases(server.Version, c) // endless

	for rel := range c {
		u.Send("availableVersion", rel)
		latest = rel
	}
}

func (u *watch) updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || latest == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	name := fmt.Sprintf(RootFS, *latest.TagName)
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
