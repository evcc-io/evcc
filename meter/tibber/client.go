package tibber

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/shurcooL/graphql"
	"golang.org/x/exp/slices"
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

func (c *Client) Homes() ([]Home, error) {
	var res struct {
		Viewer struct {
			Homes []Home
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	if err := c.Query(ctx, &res, nil); err != nil {
		return nil, err
	}

	return res.Viewer.Homes, nil
}

func (c *Client) DefaultHome(id string) (Home, error) {
	homes, err := c.Homes()
	if err != nil {
		return Home{}, err
	}

	idx := slices.IndexFunc(homes, func(h Home) bool {
		return h.ID == id || (id == "" && len(homes) == 1)
	})

	if idx == -1 {
		return Home{}, fmt.Errorf("could not determine home id: %v", homes)
	}

	return homes[idx], nil
}
