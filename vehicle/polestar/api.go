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
		CarTelemetryData `graphql:"carTelematicsV2(vins: $vins)"`
	}

	err := v.client.Query(ctx, &res, map[string]any{
		"vins": []string{vin},
	}, graphql.OperationName("CarTelematicsV2"))

	// Filter data for the requested VIN
	var filteredData CarTelemetryData

	// Filter health data
	for _, health := range res.CarTelemetryData.Health {
		if health.VIN == vin {
			filteredData.Health = append(filteredData.Health, health)
		}
	}

	// Filter battery data
	for _, battery := range res.CarTelemetryData.Battery {
		if battery.VIN == vin {
			filteredData.Battery = append(filteredData.Battery, battery)
		}
	}

	// Filter odometer data
	for _, odometer := range res.CarTelemetryData.Odometer {
		if odometer.VIN == vin {
			filteredData.Odometer = append(filteredData.Odometer, odometer)
		}
	}

	return filteredData, err
}
