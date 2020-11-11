package updater

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/go-version"
	latest "github.com/tcnksm/go-latest"
)

var instance *updater

type updater struct {
	log   *util.Logger
	cache chan<- util.Param
}

func tag2semver(tag string) string {
	if strings.Count(tag, ".") < 2 {
		tag += ".0"
	}
	return tag
}

func semver2tag(ver string) string {
	return strings.TrimSuffix(ver, ".0")
}

// Run regularly checks version
func Run(log *util.Logger, cache chan<- util.Param) {
	instance = &updater{
		log:   log,
		cache: cache,
	}

	for range time.NewTicker(time.Hour).C {
		instance.checkVersion()
	}
}

// checkVersion validates if updates are available
func (u *updater) checkVersion() {
	githubTag := &latest.GithubTag{
		Owner:      server.Owner,
		Repository: server.Repository,
	}

	installed := tag2semver(server.Version)
	if res, err := latest.Check(githubTag, installed); err == nil {
		tag := semver2tag(res.Current)
		if res.Outdated {
			u.log.INFO.Printf("new version available - please upgrade to %s", tag)
		}

		u.cache <- util.Param{
			Key: "availableVersion",
			Val: tag,
		}

		u.fetchReleaseNotes()
	} else {
		u.log.ERROR.Println("version check failed:", err)
	}
}

// fetchReleaseNotes retrieves release notes up to semver and sends to client
func (u *updater) fetchReleaseNotes() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if notes, err := releaseNotes(ctx, tag2semver(server.Version)); err == nil {
		u.cache <- util.Param{
			Key: "releaseNotes",
			Val: notes,
		}
	} else {
		u.log.WARN.Printf("couldn't download release notes: %v", err)
	}
}

// releaseNotes returns github release notes for the (from,to] semver interval
func releaseNotes(ctx context.Context, from string) (rendered string, err error) {
	var fromVersion *version.Version
	if fromVersion, err = version.NewVersion(from); err != nil {
		return
	}

	client := github.NewClient(nil)
	releases, _, err := client.Repositories.ListReleases(ctx, server.Owner, server.Repository, nil)
	if err != nil {
		return
	}

	history := bytes.NewBuffer([]byte{})
	for _, rel := range releases {
		tag := *rel.TagName

		var ver *version.Version
		if ver, err = version.NewVersion(tag2semver(tag)); err != nil {
			return
		}

		if ver.LessThanOrEqual(fromVersion) {
			continue
		}

		if notes := strings.TrimSpace(rel.GetBody()); notes != "" {
			var md string
			if md, _, err = client.Markdown(context.Background(), notes, &github.MarkdownOptions{
				Mode:    "gfm",
				Context: fmt.Sprintf("%s/%s", server.Owner, server.Repository),
			}); err != nil {
				return
			}

			history.WriteString(fmt.Sprintf("<h1>%s</h1>\n", tag))
			history.WriteString(md)
		}
	}

	return history.String(), nil
}
