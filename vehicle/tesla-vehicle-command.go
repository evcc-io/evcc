package vehicle

import (
	"context"
	"os"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	vc "github.com/evcc-io/evcc/vehicle/tesla-vehicle-command"
	"github.com/teslamotors/vehicle-command/pkg/cache"
	"github.com/teslamotors/vehicle-command/pkg/protocol"
	"golang.org/x/oauth2"
)

// TeslaVC is an api.Vehicle implementation for Tesla cars.
// It uses the official Tesla vehicle-command api.
type TeslaVC struct {
	*embed
	dataG func() (*vc.VehicleData, error)
}

var (
	TeslaClientID     string
	TeslaClientSecret string = os.Getenv("TESLA_CLIENT_SECRET")
)

func init() {
	if id := os.Getenv("TESLA_CLIENT_ID"); id != "" {
		TeslaClientID = id
	}
	if TeslaClientID != "" {
		vc.OAuth2Config.ClientID = TeslaClientID
		registry.Add("tesla-vehicle-command", NewTeslaVCFromConfig)
	}
}

const (
	privateKeyFile = "tesla-privatekey.pem"
)

// NewTeslaVCFromConfig creates a new vehicle
func NewTeslaVCFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed    `mapstructure:",squash"`
		ClientID string
		Tokens   Tokens
		VIN      string
		Timeout  time.Duration
		Cache    time.Duration
	}{
		Timeout: 10 * time.Second,
		Cache:   interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.Tokens.Error(); err != nil {
		return nil, err
	}

	if cc.ClientID != "" {
		vc.OAuth2Config.ClientID = cc.ClientID
	}

	log := util.NewLogger("tesla-vc")
	client := request.NewClient(log)

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	ts := vc.OAuth2Config.TokenSource(ctx, &oauth2.Token{
		AccessToken:  cc.Tokens.Access,
		RefreshToken: cc.Tokens.Refresh,
		Expiry:       time.Now(),
	})

	identity, err := vc.NewIdentity(log, ts)
	if err != nil {
		return nil, err
	}

	vcapi := vc.NewAPI(log, identity, cc.Timeout)

	vehicle, err := ensureVehicleEx(
		cc.VIN, vcapi.Vehicles,
		func(v *vc.Vehicle) string {
			return v.Vin
		},
	)
	if err != nil {
		return nil, err
	}

	v := &TeslaVC{
		embed: &cc.embed,
		dataG: provider.Cached(func() (*vc.VehicleData, error) {
			res, err := vcapi.VehicleData(vehicle.ID)
			return res, vc.ApiError(err)
		}, cc.Cache),
	}

	if v.Title_ == "" {
		v.Title_ = vehicle.DisplayName
	}

	privKey, err := protocol.LoadPrivateKey(privateKeyFile)
	if err != nil {
		log.WARN.Println("private key not found, commands are disabled")
		return v, nil
	}

	vv, err := identity.Account().GetVehicle(context.Background(), vehicle.Vin, privKey, cache.New(8))
	if err != nil {
		return nil, err
	}

	cs, err := vc.NewCommandSession(vv, cc.Timeout)
	if err != nil {
		return nil, err
	}

	res := &struct {
		*TeslaVC
		*vc.CommandSession
	}{
		TeslaVC:        v,
		CommandSession: cs,
	}

	return res, nil
}

// Soc implements the api.Vehicle interface
func (v *TeslaVC) Soc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.UsableBatteryLevel), nil
}

var _ api.ChargeState = (*TeslaVC)(nil)

// Status implements the api.ChargeState interface
func (v *TeslaVC) Status() (api.ChargeStatus, error) {
	status := api.StatusA // disconnected
	res, err := v.dataG()
	if err != nil {
		return status, err
	}

	switch res.Response.ChargeState.ChargingState {
	case "Stopped", "NoPower", "Complete":
		status = api.StatusB
	case "Charging":
		status = api.StatusC
	}

	return status, nil
}

var _ api.ChargeRater = (*TeslaVC)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *TeslaVC) ChargedEnergy() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return res.Response.ChargeState.ChargeEnergyAdded, nil
}

var _ api.VehicleRange = (*TeslaVC)(nil)

// Range implements the api.VehicleRange interface
func (v *TeslaVC) Range() (int64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	// miles to km
	return int64(kmPerMile * res.Response.ChargeState.BatteryRange), nil
}

var _ api.VehicleOdometer = (*TeslaVC)(nil)

// Odometer implements the api.VehicleOdometer interface
func (v *TeslaVC) Odometer() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	// miles to km
	return kmPerMile * res.Response.VehicleState.Odometer, nil
}

var _ api.VehicleFinishTimer = (*TeslaVC)(nil)

// FinishTime implements the api.VehicleFinishTimer interface
func (v *TeslaVC) FinishTime() (time.Time, error) {
	res, err := v.dataG()
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(time.Duration(res.Response.ChargeState.MinutesToFullCharge) * time.Minute), nil
}

// TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

// var _ api.VehiclePosition = (*TeslaVC)(nil)

// // Position implements the api.VehiclePosition interface
// func (v *TeslaVC) Position() (float64, float64, error) {
// 	res, err := v.dataG()
// 	if err != nil {
// 		return 0, 0, err
// 	}
// 	if res.Response.DriveState.Latitude != 0 || res.Response.DriveState.Longitude != 0 {
// 		return res.Response.DriveState.Latitude, res.Response.DriveState.Longitude, nil
// 	}
// 	return res.Response.DriveState.ActiveRouteLatitude, res.Response.DriveState.ActiveRouteLongitude, nil
// }

var _ api.SocLimiter = (*TeslaVC)(nil)

// TargetSoc implements the api.SocLimiter interface
func (v *TeslaVC) TargetSoc() (float64, error) {
	res, err := v.dataG()
	if err != nil {
		return 0, err
	}
	return float64(res.Response.ChargeState.ChargeLimitSoc), nil
}
