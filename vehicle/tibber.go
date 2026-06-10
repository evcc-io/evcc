package vehicle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/tibber"
)

// Tibber is an api.Vehicle implementation for the Tibber Data API
type Tibber struct {
	*embed
	*tibber.API
	vin    string
	homeID string
	devID  string
	dataG  func() (tibber.DeviceDetail, error)
}

func init() {
	registry.AddCtx("tibber", NewTibberFromConfig)
}

// NewTibberFromConfig creates a new vehicle
func NewTibberFromConfig(ctx context.Context, other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed                               `mapstructure:",squash"`
		ClientID, ClientSecret, RedirectURI string
		VIN                                 string
		Cache                               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.ClientID == "" || cc.ClientSecret == "" {
		return nil, api.ErrMissingCredentials
	}

	log := util.NewLogger("tibber").Redact(cc.ClientID, cc.ClientSecret)

	authCtx := util.WithLogger(context.Background(), log)
	ts, err := tibber.NewOAuth(authCtx, cc.ClientID, cc.ClientSecret, cc.RedirectURI, cc.embed.GetTitle())
	if err != nil {
		return nil, err
	}

	v := &Tibber{
		embed: &cc.embed,
		API:   tibber.NewAPI(log, ts),
		vin:   strings.ToUpper(cc.VIN),
	}

	v.dataG = util.Cached(v.status, cc.Cache)

	return v, nil
}

// resolve discovers the home and device id of the configured vehicle. It is
// deferred until first use because authorization happens interactively. With
// only the vehicles scope granted, the devices endpoint returns vehicles only.
func (v *Tibber) resolve() error {
	homes, err := v.Homes()
	if err != nil {
		return err
	}

	for _, home := range homes {
		devices, err := v.Devices(home.ID)
		if err != nil {
			return err
		}

		for _, d := range devices {
			// match the bare vin or the full external id (e.g. tesla:5YJ...)
			if v.vin == "" || strings.EqualFold(d.VIN(), v.vin) || strings.EqualFold(d.ExternalID, v.vin) {
				v.homeID = home.ID
				v.devID = d.ID
				return nil
			}
		}
	}

	return fmt.Errorf("vehicle not found: %s", v.vin)
}

func (v *Tibber) status() (tibber.DeviceDetail, error) {
	if v.devID == "" {
		if err := v.resolve(); err != nil {
			return tibber.DeviceDetail{}, err
		}
	}

	return v.Device(v.homeID, v.devID)
}

// Soc implements the api.Vehicle interface
func (v *Tibber) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}

	soc, ok := res.Soc()
	if !ok {
		return 0, api.ErrNotAvailable
	}

	return soc, nil
}

var _ api.ChargeState = (*Tibber)(nil)

// Status implements the api.ChargeState interface
func (v *Tibber) Status() (api.ChargeStatus, error) {
	res, err := v.dataG()
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusA // disconnected

	if plug, ok := res.PlugStatus(); ok && plug == tibber.StatusConnected {
		status = api.StatusB // connected, not charging
	}
	if charging, ok := res.ChargingStatus(); ok && charging == tibber.StatusCharging {
		status = api.StatusC // charging
	}

	return status, nil
}

var _ api.SocLimiter = (*Tibber)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (v *Tibber) GetLimitSoc() (int64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}

	soc, ok := res.TargetSoc()
	if !ok {
		return 0, api.ErrNotAvailable
	}

	return int64(soc), nil
}

var _ api.VehicleRange = (*Tibber)(nil)

// Range implements the api.VehicleRange interface
func (v *Tibber) Range() (int64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}

	rng, ok := res.Range()
	if !ok {
		return 0, api.ErrNotAvailable
	}

	return int64(rng), nil
}
