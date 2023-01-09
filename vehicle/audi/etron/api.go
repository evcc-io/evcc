package etron

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

const ApiURI = "https://app-api.live-my.audi.com/vgql/v1/graphql"

// API is the VW api client
type API struct {
	client *graphql.Client
}

// NewAPI creates a new api client
func NewAPI(log *util.Logger, ts oauth2.TokenSource) *API {
	ctx := context.WithValue(
		context.Background(),
		oauth2.HTTPClient,
		request.NewClient(log),
	)

	v := &API{
		client: graphql.NewClient(ApiURI, oauth2.NewClient(ctx, ts)),
	}

	return v
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles(ctx context.Context) ([]Vehicle, error) {
	var res struct {
		UserVehicles []Vehicle
	}

	err := v.client.Query(ctx, &res, nil)

	return res.UserVehicles, err
}
