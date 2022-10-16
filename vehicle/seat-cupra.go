package vehicle

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/seat"
	"github.com/evcc-io/evcc/vehicle/seat/cupra"
	"github.com/evcc-io/evcc/vehicle/vag/service"
	"github.com/evcc-io/evcc/vehicle/vag/tokenrefreshservice"
	"github.com/evcc-io/evcc/vehicle/vag/vwidentity"
	"golang.org/x/oauth2"
)

// Cupra is an api.Vehicle implementation for Seat Cupra cars
type Cupra struct {
	*embed
	*cupra.Provider // provides the api implementations
}

func init() {
	registry.AddWithStore("cupra", NewCupraFromConfig)
}

// NewCupraFromConfig creates a new vehicle
func NewCupraFromConfig(factory store.Provider, other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Cupra{
		embed: &cc.embed,
	}

	log := util.NewLogger("cupra").Redact(cc.User, cc.Password, cc.VIN)

	trsStore := factory("seat.tokens.trs." + cc.User)
	trs := tokenrefreshservice.New(log, seat.TRSParams).WithStore(trsStore)

	ts, err := service.TokenRefreshServiceTokenSource(log, trs, seat.AuthParams, cc.User, cc.Password)
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

	cc.VIN, err = ensureVehicle(cc.VIN, func() ([]string, error) {
		return api.Vehicles(ui.Subject)
	})

	if err == nil {
		v.Provider = cupra.NewProvider(api, ui.Subject, cc.VIN, cc.Cache)
	}

	return v, err
}
