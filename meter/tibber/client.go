package tibber

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

type Client struct {
	*graphql.Client
}

func NewClient(log *util.Logger, token string) *Client {
	ctx := context.WithValue(
		context.Background(),
		oauth2.HTTPClient,
		request.NewHelper(log).Client,
	)

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	}))

	return &Client{
		Client: graphql.NewClient(URI, client),
	}
}

func (c *Client) Home() (Home, error) {
	var res struct {
		Viewer struct {
			Homes []Home
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	if err := c.Query(ctx, &res, nil); err != nil {
		return Home{}, err
	}

	if len(res.Viewer.Homes) != 1 {
		return Home{}, fmt.Errorf("could not determine home id: %v", res.Viewer.Homes)
	}

	return res.Viewer.Homes[0], nil
}

func (c *Client) DefaultHomeID() (string, error) {
	home, err := c.Home()
	return home.ID, err
}
