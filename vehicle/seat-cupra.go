package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/seat/cupra"
	"github.com/evcc-io/evcc/vehicle/vag/service"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

// Cupra is an api.Vehicle implementation for Seat Cupra cars
type Cupra struct {
	*embed
	*cupra.Provider // provides the api implementations
}

func init() {
	registry.Add("cupra", NewCupraFromConfig)
}

// NewCupraFromConfig creates a new vehicle
func NewCupraFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Cupra{
		embed: &cc.embed,
	}

	log := util.NewLogger("cupra").Redact(cc.User, cc.Password, cc.VIN)

	ts, err := service.TokenRefreshServiceTokenSource(log, cupra.TRSParams, cupra.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	// get OIDC user information
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, request.NewClient(log))
	ui, err := vwidentity.Config.NewProvider(ctx).UserInfo(ctx, ts)
	if err != nil {
		return nil, fmt.Errorf("failed getting user information: %w", err)
	}

	api := cupra.NewAPI(log, ts)

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
		v.Provider = cupra.NewProvider(api, ui.Subject, vehicle.VIN, cc.Cache)
	}

	return v, err
}
