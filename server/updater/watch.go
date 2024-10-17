package updater

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/google/go-github/v32/github"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-version"
)

type webServer interface {
	Router() *mux.Router
}

type watch struct {
	log     *util.Logger
	outChan chan<- util.Param
	repo    *Repo
}

func (u *watch) Send(key string, val interface{}) {
	u.outChan <- util.Param{
		Key: key,
		Val: val,
	}
}

func (u *watch) watchReleases(installed string, out chan *github.RepositoryRelease) {
	tick := time.NewTicker(6 * time.Hour)
	for ; true; <-tick.C {
		rel, err := u.findReleaseUpdate(installed)
		if err != nil {
			u.log.ERROR.Printf("version check failed: %v (installed: %s)", err, installed)
			continue
		}

		if rel != nil {
			u.log.INFO.Printf("new version available: %s", *rel.TagName)
			out <- rel
		}
	}
}

// findReleaseUpdate validates if updates are available
func (u *watch) findReleaseUpdate(installed string) (*github.RepositoryRelease, error) {
	rel, err := u.repo.GetLatestRelease()
	if err != nil {
		return nil, err
	}

	if rel.TagName == nil {
		return nil, errors.New("untagged release")
	}

	v1, err := version.NewVersion(installed)
	if err != nil {
		return nil, err
	}

	v2, err := version.NewVersion(*rel.TagName)
	if err != nil {
		return nil, err
	}

	if v1.LessThan(v2) {
		go u.fetchReleaseNotes(installed)
		return rel, nil
	}

	// no update
	return nil, nil
}

// fetchReleaseNotes retrieves release notes up to semver and sends to client
func (u *watch) fetchReleaseNotes(installed string) {
	if notes, err := u.repo.ReleaseNotes(installed); err == nil {
		u.outChan <- util.Param{
			Key: "releaseNotes",
			Val: notes,
		}
	} else {
		u.log.WARN.Printf("couldn't download release notes: %v", err)
	}
}
