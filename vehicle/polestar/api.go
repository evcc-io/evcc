package polestar

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

// https://github.com/TA2k/ioBroker.polestar

const ApiURI = "https://pc-api.polestar.com/eu-north-1/my-star"

type API struct {
	*request.Helper
	client *graphql.Client
}

func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	httpClient := request.NewClient(log)

	// replace client transport with authenticated transport
	httpClient.Transport = &oauth2.Transport{
		Base:   httpClient.Transport,
		Source: identity,
	}

	v := &API{
		Helper: request.NewHelper(log),
		client: graphql.NewClient(ApiURI, httpClient),
	}

	v.Client.Transport = httpClient.Transport

	return v
}

func (v *API) Vehicles(ctx context.Context) ([]ConsumerCar, error) {
	var res struct {
		GetConsumerCarsV2 []ConsumerCar `graphql:"getConsumerCarsV2"`
	}

	err := v.client.Query(ctx, &res, nil, graphql.OperationName("getCars"))
	return res.GetConsumerCarsV2, err
}

func (v *API) Status(ctx context.Context, vin string) (BatteryData, error) {
	var res struct {
		BatteryData `graphql:"getBatteryData(vin: $vin)"`
	}

	err := v.client.Query(ctx, &res, map[string]interface{}{
		"vin": vin,
	}, graphql.OperationName("GetBatteryData"))
	return res.BatteryData, err
}

func (v *API) Odometer(ctx context.Context, vin string) (OdometerData, error) {
	var res struct {
		OdometerData `graphql:"getOdometerData(vin: $vin)"`
	}

	err := v.client.Query(ctx, &res, map[string]interface{}{
		"vin": vin,
	}, graphql.OperationName("GetOdometerData"))
	return res.OdometerData, err
}
