// +build gokrazy

package updater

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/google/go-github/v32/github"
)

const (
	RootFS = "evcc_%s.rootfs.gz"
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

	// ssh handler
	jar, _ := cookiejar.New(nil)
	client = &http.Client{Jar: jar}
	httpd.Router().PathPrefix("/api/ssh").HandlerFunc(u.sshHandler)

	// update handler
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
