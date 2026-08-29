package tesla

import (
	"context"
	"errors"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	teslaclient "github.com/evcc-io/tesla-proxy-client"
	"github.com/teslamotors/vehicle-command/pkg/account"
	"github.com/teslamotors/vehicle-command/pkg/cache"
	"github.com/teslamotors/vehicle-command/pkg/protocol"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
	"golang.org/x/oauth2"
)

// SignedController sends commands via the Vehicle Command Protocol, signed with the instance key.
// Vehicles without protocol support (pre-2021 Model S/X) fall back to the REST controller.
type SignedController struct {
	mu       sync.Mutex
	ts       oauth2.TokenSource
	key      protocol.ECDHPrivateKey
	vin      string
	host     string
	sessions *cache.SessionCache
	rest     *Controller
}

// NewSignedController creates a signing vehicle current and charge controller
func NewSignedController(ts oauth2.TokenSource, key protocol.ECDHPrivateKey, vehicle *teslaclient.Vehicle, host string) *SignedController {
	return &SignedController{
		ts:       ts,
		key:      key,
		vin:      vehicle.Vin,
		host:     host,
		sessions: cache.New(1),
		rest:     NewController(vehicle),
	}
}

func (v *SignedController) exec(fn func(context.Context, *vehicle.Vehicle) error) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	token, err := v.ts.Token()
	if err != nil {
		return err
	}

	acct, err := account.New(token.AccessToken, "evcc/"+util.Version)
	if err != nil {
		return err
	}
	acct.Host = v.host

	car, err := acct.GetVehicle(ctx, v.vin, v.key, v.sessions)
	if err != nil {
		return err
	}

	if err := car.Connect(ctx); err != nil {
		return err
	}
	defer car.Disconnect()

	if err := car.StartSession(ctx, nil); err != nil {
		return err
	}
	defer func() { _ = car.UpdateCachedSessions(v.sessions) }()

	err = fn(ctx, car)

	// command not applicable in current vehicle state, e.g. already charging
	if protocol.IsNominalError(err) {
		err = nil
	}

	return err
}

var _ api.CurrentController = (*SignedController)(nil)

// MaxCurrent implements the api.CurrentController interface
func (v *SignedController) MaxCurrent(current int64) error {
	err := v.exec(func(ctx context.Context, car *vehicle.Vehicle) error {
		return car.SetChargingAmps(ctx, int32(current))
	})

	if errors.Is(err, protocol.ErrProtocolNotSupported) {
		return v.rest.MaxCurrent(current)
	}

	return apiError(err)
}

var _ api.ChargeController = (*SignedController)(nil)

// ChargeEnable implements the api.ChargeController interface
func (v *SignedController) ChargeEnable(enable bool) error {
	err := v.exec(func(ctx context.Context, car *vehicle.Vehicle) error {
		if enable {
			return car.ChargeStart(ctx)
		}
		return car.ChargeStop(ctx)
	})

	if errors.Is(err, protocol.ErrProtocolNotSupported) {
		return v.rest.ChargeEnable(enable)
	}

	err = apiError(err)

	// ignore sleeping vehicle
	if !enable && errors.Is(err, api.ErrAsleep) {
		err = nil
	}

	return err
}
