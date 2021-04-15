package sponsor

import (
	"context"

	"github.com/andig/evcc/util"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var client *githubv4.Client

func init() {
	token := util.Getenv("GITHUB_TOKEN")

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	client = githubv4.NewClient(tc)
}

// Source: https://github.com/tj/sponsors-api/blob/master/server.go

// Sponsor model
type Sponsor struct {
	Name      string
	Login     string
	AvatarURL string
}

// sponsorships query: https://docs.github.com/en/graphql/reference/objects
type sponsorships struct {
	Viewer struct {
		Login                    string
		SponsorshipsAsMaintainer struct {
			PageInfo struct {
				EndCursor   string
				HasNextPage bool
			}

			Edges []struct {
				Node struct {
					Sponsor Sponsor
				}
				Cursor string
			}
		} `graphql:"sponsorshipsAsMaintainer(first: 100, after: $cursor, includePrivate: true)"`
	}
}

// Get returns the list of sponsors
func Get(ctx context.Context) ([]Sponsor, error) {
	var sponsors []Sponsor
	var q sponsorships
	var cursor string

	for {
		err := client.Query(ctx, &q, map[string]interface{}{
			"cursor": githubv4.String(cursor),
		})

		if err != nil {
			return nil, err
		}

		for _, edge := range q.Viewer.SponsorshipsAsMaintainer.Edges {
			sponsor := edge.Node.Sponsor
			sponsors = append(sponsors, sponsor)
		}

		if !q.Viewer.SponsorshipsAsMaintainer.PageInfo.HasNextPage {
			break
		}

		cursor = q.Viewer.SponsorshipsAsMaintainer.PageInfo.EndCursor
	}

	return sponsors, nil
}
