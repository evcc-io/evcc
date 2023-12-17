package polestar

import (
	"context"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.polestar

const ApiURI = "https://pc-api.polestar.com/eu-north-1/mesh"

type API struct {
	*request.Helper
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
		Helper: request.NewHelper(log),
		client: graphql.NewClient(ApiURI, httpClient),
	}

	v.Client.Transport = httpClient.Transport

	return v
}

func (v *API) Vehicles(ctx context.Context) ([]string, error) {
	var res struct {
		GetConsumerCarsV2 struct {
			Vin                       string
			InternalVehicleIdentifier string
		}
	}

	var vins []string
	err := v.client.Query(ctx, &res, nil, graphql.OperationName("getCars"))
	// if err == nil {
	// 	vins = lo.Map(res.MyStar.GetConsumerCars, func(v ConsumerCar, _ int) string {
	// 		return v.Vin
	// 	})
	// }

	fmt.Println(res)

	// v.Status(ctx, vins[0])

	return vins, err
}

type VehicleInformation struct {
	VdmsExtendedCarDetails `graphql:"... on VehicleInformation"`
}

type VdmsExtendedCarDetails struct {
	Type string `graphql:"__typename"`
}

// {
// 	"query": "query($locale:String!$vin:String!){
// 		vdms{
// 			vehicleInformation(vin: $vin, locale: $locale){
// 				... on VehicleInformation{
// 					__typename
// 				}
// 			}
// 		}
// 	}",
// 	"variables": {
// 		"locale": "de_DE",
// 		"vin": "LPSVSEDEEML002398"
// 	}
// }

func (v *API) Status(ctx context.Context, vin string) error {
	// var res struct {
	// 	// GetVDMSCarDetails struct {
	// 	Vdms struct {
	// 		VehicleInformation `graphql:"vehicleInformation(vin: $vin, locale: $locale)"`
	// 	} //`graphql:"GetVDMSCarDetails($vin: String!, $locale: String!)"`
	// 	// }
	// }

	// err := v.client.Query(ctx, &res, map[string]interface{}{
	// 	"vin":    graphql.String(vin),
	// 	"locale": graphql.String("de_DE"),
	// })
	// if err == nil {

	// }

	err := v.GetJSON(fmt.Sprintf("%s/status/%s", ApiURI, vin), nil)
	return err
}

// func (v *API) Refresh(vin string) (StatusResponse, error) {
// 	var res StatusResponse

// 	uri := fmt.Sprintf("%s/vehicles/%s/refresh-data", ApiURI, vin)
// 	err := v.GetJSON(uri, &res)

// 	if err != nil && res.Error != "" {
// 		err = fmt.Errorf("%s (%s): %w", res.Error, res.ErrorDescription, err)
// 	}

// 	return res, err
// }
