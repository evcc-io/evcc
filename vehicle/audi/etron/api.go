package etron

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/shurcooL/graphql"
	"github.com/thoas/go-funk"
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
		request.NewHelper(log).Client,
	)

	v := &API{
		client: graphql.NewClient(ApiURI, oauth2.NewClient(ctx, ts)),
	}

	return v
}

// Vehicles implements the /vehicles response
func (v *API) Vehicles() ([]string, error) {
	type vehicle struct {
		VIN, Type, Nickname string
	}

	var res struct {
		UserVehicles []vehicle `json:"userVehicles"`
	}

	var vins []string
	err := v.client.Query(context.Background(), &res, nil)
	if err == nil {
		vins = funk.Map(res.UserVehicles, func(v vehicle) string {
			return v.VIN
		}).([]string)
	}

	return vins, err
}
