package polestar

import (
	"context"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

// https://github.com/leeyuentuen/polestar_api
// https://github.com/TA2k/ioBroker.polestar

const (
	ApiURI   = "https://pc-api.polestar.com/eu-north-1"
	ApiURIv2 = ApiURI + "/mystar-v2"
)

type API struct {
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
		client: graphql.NewClient(ApiURIv2, httpClient),
	}

	return v
}

func (v *API) Vehicles(ctx context.Context) ([]ConsumerCar, error) {
	var res struct {
		GetConsumerCarsV2 []ConsumerCar `graphql:"getConsumerCarsV2"`
	}

	err := v.client.Query(ctx, &res, nil, graphql.OperationName("getCars"))

	return res.GetConsumerCarsV2, err
}

func (v *API) CarTelemetry(ctx context.Context, vin string) (CarTelemetryData, error) {
	var res struct {
		CarTelemetryData `graphql:"carTelematics(vin: $vin)"`
	}

	err := v.client.Query(ctx, &res, map[string]any{
		"vin": vin,
	}, graphql.OperationName("CarTelematics"))

	return res.CarTelemetryData, err
}
