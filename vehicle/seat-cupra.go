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

	// get initial VW identity id_token
	q, err := vwidentity.Login(log, seat.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	trs := tokenrefreshservice.New(log, seat.TRSParams)
	token, err := trs.Exchange(q)
	if err != nil {
		return nil, err
	}

	ts := trs.TokenSource(token)

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
