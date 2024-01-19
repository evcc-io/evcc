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
	ApiURI   = "https://pc-api.polestar.com/eu-north-1/my-star"
	ApiURIv2 = "https://pc-api.polestar.com/eu-north-1/mystar-v2"
)

type API struct {
	client  *graphql.Client
	client2 *graphql.Client
}

func NewAPI(log *util.Logger, identity oauth2.TokenSource) *API {
	httpClient := request.NewClient(log)

	// replace client transport with authenticated transport
	httpClient.Transport = &oauth2.Transport{
		Base:   httpClient.Transport,
		Source: identity,
	}

	httpClient2 := request.NewClient(log)

	// replace client transport with authenticated transport
	httpClient2.Transport = &oauth2.Transport{
		Base:   httpClient2.Transport,
		Source: identity,
	}

	v := &API{
		client:  graphql.NewClient(ApiURI, httpClient),
		client2: graphql.NewClient(ApiURIv2, httpClient2),
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

func (v *API) Status(ctx context.Context, vin string) (BatteryData, error) {
	var res struct {
		BatteryData `graphql:"getBatteryData(vin: $vin)"`
	}

	// err := v.client.WithRequestModifier(func(req *http.Request) {
	// 	var payload graphql.GraphQLRequestPayload
	// 	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
	// 		panic(err)
	// 	}
	// 	vars, err := json.Marshal(payload.Variables)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	req.URL, err = url.Parse(fmt.Sprintf("%s?query=%s&variables=%s&operationName=%s", ApiURIv2,
	// 		url.PathEscape(payload.Query), url.PathEscape(string(vars)), url.PathEscape(payload.OperationName)))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	req.Method = http.MethodGet
	// 	req.Body = nil
	// }).Query(ctx, &res, map[string]any{
	err := v.client2.Query(ctx, &res, map[string]any{
		"vin": vin,
	}, graphql.OperationName("GetBatteryData"))

	// os.Exit(1)
	return res.BatteryData, err
}

func (v *API) Odometer(ctx context.Context, vin string) (OdometerData, error) {
	var res struct {
		OdometerData `graphql:"getOdometerData(vin: $vin)"`
	}

	err := v.client2.Query(ctx, &res, map[string]any{
		"vin": vin,
	}, graphql.OperationName("GetOdometerData"))

	return res.OdometerData, err
}
