package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/seat"
	"github.com/evcc-io/evcc/vehicle/seat/cupra"
	"github.com/evcc-io/evcc/vehicle/vag/service"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"github.com/evcc-io/evcc/vehicle/vw"
	"golang.org/x/oauth2"
)

// https://github.com/trocotronic/weconnect
// https://github.com/TA2k/ioBroker.vw-connect

// Seat is an api.Vehicle implementation for Seat cars
type Seat struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("seat", NewSeatFromConfig)
}

// NewSeatFromConfig creates a new vehicle
func NewSeatFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &Seat{
		embed: &cc.embed,
	}

	log := util.NewLogger("seat").Redact(cc.User, cc.Password, cc.VIN)

	trs, err := service.TokenRefreshServiceTokenSource(log, seat.TRSParams, seat.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	// get OIDC user information
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))
	ui, err := vwidentity.Config.NewProvider(ctx).UserInfo(ctx, trs)
	if err != nil {
		return nil, fmt.Errorf("failed getting user information: %w", err)
	}

	api := cupra.NewAPI(log, trs)

	vehicle, err := ensureVehicleEx(
		cc.VIN, func() ([]cupra.Vehicle, error) {
			return api.Vehicles(ui.Subject)
		},
		func(v cupra.Vehicle) string {
			return v.VIN
		},
	)

	if err == nil {
		v.fromVehicle(vehicle.VehicleNickname, 0)
	}

	if err == nil {
		ts := service.MbbTokenSource(log, trs, seat.AuthClientID)
		api := vw.NewAPI(log, ts, seat.Brand, seat.Country)
		api.Client.Timeout = cc.Timeout

		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, vehicle.VIN, cc.Cache)
		}
	}

	return v, err
}
