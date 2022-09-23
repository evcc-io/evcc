package polestar

import (
	"context"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.polestar

const ApiURI = "https://pc-api.polestar.com/eu-north-1/mesh/"

type API struct {
	client *graphql.Client
}

func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	httpClient := request.NewClient(log)

	// replace client transport with authenticated transport
	httpClient.Transport = &transport.Decorator{
		Base: httpClient.Transport,
		Decorator: func(req *http.Request) error {
			token, err := identity.Token()
			if err == nil {
				req.Header.Set("x-polestarid-authorization", "Bearer "+token.AccessToken)
			}
			return err
		},
	}

	v := &API{
		client: graphql.NewClient(ApiURI, httpClient),
	}

	return v
}

func (v *API) Vehicles(ctx context.Context) ([]string, error) {
	type vehicle struct {
		VIN, Type, Nickname string
	}

	var res struct {
		GetConsumerInformation struct {
			MyStar struct {
				Type        string `graphql:"__typename"`
				GetConsumer struct {
					Type                  string `graphql:"__typename"`
					MyStarConsumerDetails string `graphql:"... MyStarConsumerDetails"`
				} `graphql:"getConsumer"`
			} `graphql:"myStar"`
		} `graphql:"GetConsumerInformation"`
	}

	var vins []string
	err := v.client.Query(ctx, &res, nil)
	// if err == nil {
	// 	vins = lo.Map(res.UserVehicles, func(v vehicle, _ int) string {
	// 		return v.VIN
	// 	})
	// }

	return vins, err
}

// func (v *API) Status(vin string) (StatusResponse, error) {
// 	var res StatusResponse

// 	uri := fmt.Sprintf("%s/vehicles/%s/init-data?requestedData=BOTH&countryCode=DE&locale=de-DE", ApiURI, vin)
// 	err := v.GetJSON(uri, &res)

// 	if err != nil && res.Error != "" {
// 		err = fmt.Errorf("%s (%s): %w", res.Error, res.ErrorDescription, err)
// 	}

// 	return res, err
// }

// func (v *API) Refresh(vin string) (StatusResponse, error) {
// 	var res StatusResponse

// 	uri := fmt.Sprintf("%s/vehicles/%s/refresh-data", ApiURI, vin)
// 	err := v.GetJSON(uri, &res)

// 	if err != nil && res.Error != "" {
// 		err = fmt.Errorf("%s (%s): %w", res.Error, res.ErrorDescription, err)
// 	}

// 	return res, err
// }
