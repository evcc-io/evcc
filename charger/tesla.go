package charger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bogosj/tesla"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle"
	"golang.org/x/oauth2"
)

// Tesla is an api.Vehicle implementation for Tesla cars
type Tesla struct {
	*request.Helper
	uri           string
	vitalsG       func() (Vitals, error)
	vehicle       *tesla.Vehicle
	chargeStateG  func() (*tesla.ChargeState, error)
	vehicleStateG func() (*tesla.VehicleState, error)
	driveStateG   func() (*tesla.DriveState, error)
	enabled       bool
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

// Vitals is the /api/1/vitals response
type Vitals struct {
	ContactorClosed   bool    `json:"contactor_closed"`    //false
	VehicleConnected  bool    `json:"vehicle_connected"`   //false
	SessionS          float64 `json:"session_s"`           //0
	GridV             float64 `json:"grid_v"`              //230.1
	GridHz            float64 `json:"grid_hz"`             //49.928
	VehicleCurrentA   float64 `json:"vehicle_current_a"`   //0.1
	CurrentAA         float64 `json:"currentA_a"`          //0.0
	CurrentBA         float64 `json:"currentB_a"`          //0.1
	CurrentCA         float64 `json:"currentC_a"`          //0.0
	CurrentNA         float64 `json:"currentN_a"`          //0.0
	VoltageAV         float64 `json:"voltageA_v"`          //0.0
	VoltageBV         float64 `json:"voltageB_v"`          //0.0
	VoltageCV         float64 `json:"voltageC_v"`          //0.0
	RelayCoilV        float64 `json:"relay_coil_v"`        //11.8
	PcbaTempC         float64 `json:"pcba_temp_c"`         //19.2
	HandleTempC       float64 `json:"handle_temp_c"`       //15.3
	McuTempC          float64 `json:"mcu_temp_c"`          //25.1
	UptimeS           int     `json:"uptime_s"`            //831580
	InputThermopileUv float64 `json:"input_thermopile_uv"` //-233
	ProxV             float64 `json:"prox_v"`              //0.0
	PilotHighV        float64 `json:"pilot_high_v"`        //11.9
	PilotLowV         float64 `json:"pilot_low_v"`         //11.9
	SessionEnergyWh   float64 `json:"session_energy_wh"`   //22864.699
	ConfigStatus      int     `json:"config_status"`       //5
	EvseState         int     `json:"evse_state"`          //1
	CurrentAlerts     []any   `json:"current_alerts"`      //[]
}

// NewTeslaFromConfig creates a new vehicle
func NewTeslaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI    string
		Tokens vehicle.Tokens
		VIN    string
		Cache  time.Duration
	}{
		Cache: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if err := cc.Tokens.Error(); err != nil {
		return nil, err
	}

	log := util.NewLogger("tesla").Redact(cc.Tokens.Access, cc.Tokens.Refresh)

	c := &Tesla{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(cc.URI, "http"),
	}

	c.vitalsG = provider.Cached(func() (Vitals, error) {
		uri := fmt.Sprintf("%s/api/1/vitals ", c.uri)
		var res Vitals
		err := c.GetJSON(uri, &res)
		return res, err
	}, time.Second)

	// authenticated http client with logging injected to the Tesla client
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c.Helper.Client)

	options := []tesla.ClientOption{tesla.WithToken(&oauth2.Token{
		AccessToken:  cc.Tokens.Access,
		RefreshToken: cc.Tokens.Refresh,
		Expiry:       time.Now(),
	})}

	client, err := tesla.NewClient(ctx, options...)
	if err != nil {
		return nil, err
	}

	vv, err := client.Vehicles()
	if err != nil {
		return nil, err
	}

	for _, v := range vv {
		if strings.EqualFold(v.Vin, cc.VIN) {
			c.vehicle = v
			break
		}
	}

	if c.vehicle == nil {
		return nil, fmt.Errorf("Tesla %s not found", cc.VIN)
	}

	c.chargeStateG = provider.Cached(c.vehicle.ChargeState, cc.Cache)
	c.vehicleStateG = provider.Cached(c.vehicle.VehicleState, cc.Cache)
	c.driveStateG = provider.Cached(c.vehicle.DriveState, cc.Cache)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Tesla) Enabled() (bool, error) {
	return c.enabled, nil
}

// Enable implements the api.Charger interface
func (c *Tesla) Enable(enable bool) error {
	var err error
	if enable {
		err = c.vehicle.StartCharging()
	} else {
		err = c.vehicle.StopCharging()
	}

	if err == nil {
		c.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Tesla) MaxCurrent(current int64) error {
	return c.vehicle.SetChargingAmps(int(current))
}

// Status implements the api.Charger interface
func (v *Tesla) Status() (api.ChargeStatus, error) {
	{
		// check TWC status first
		res, err := v.vitalsG()
		if !res.VehicleConnected || err != nil {
			return api.StatusA, err
		}
	}

	status := api.StatusA // disconnected
	res, err := v.chargeStateG()

	if err == nil {
		if res.ChargingState == "Stopped" || res.ChargingState == "NoPower" || res.ChargingState == "Complete" {
			status = api.StatusB
		}
		if res.ChargingState == "Charging" {
			status = api.StatusC
		}
	}

	return status, err
}

var _ api.ChargeRater = (*Tesla)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (v *Tesla) ChargedEnergy() (float64, error) {
	res, err := v.vitalsG()
	return res.SessionEnergyWh / 1e3, err
}

var _ api.MeterCurrent = (*Tesla)(nil)

// Currents implements the api.MeterCurrent interface
func (v *Tesla) Currents() (float64, float64, float64, error) {
	res, err := v.vitalsG()
	return res.CurrentAA, res.CurrentBA, res.CurrentCA, err
}

// var _ api.MeterEnergy = (*Tesla)(nil)

// // TotalEnergy implements the api.MeterEnergy interface
// func (c *Tesla) TotalEnergy() (float64, error) {
// 	return 1, nil
// }

// // SoC implements the api.Vehicle interface
// func (v *Tesla) SoC() (float64, error) {
// 	res, err := v.chargeStateG()

// 	if err == nil {
// 		return float64(res.UsableBatteryLevel), nil
// 	}

// 	return 0, err
// }

// const kmPerMile = 1.609344

// var _ api.VehicleRange = (*Tesla)(nil)

// // Range implements the api.VehicleRange interface
// func (v *Tesla) Range() (int64, error) {
// 	res, err := v.chargeStateG()

// 	if err == nil {
// 		// miles to km
// 		return int64(kmPerMile * res.BatteryRange), nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleOdometer = (*Tesla)(nil)

// // Odometer implements the api.VehicleOdometer interface
// func (v *Tesla) Odometer() (float64, error) {
// 	res, err := v.vehicleStateG()

// 	if err == nil {
// 		// miles to km
// 		return kmPerMile * res.Odometer, nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleFinishTimer = (*Tesla)(nil)

// // FinishTime implements the api.VehicleFinishTimer interface
// func (v *Tesla) FinishTime() (time.Time, error) {
// 	res, err := v.chargeStateG()

// 	if err == nil {
// 		t := time.Now()
// 		return t.Add(time.Duration(res.MinutesToFullCharge) * time.Minute), err
// 	}

// 	return time.Time{}, err
// }

// // TODO api.Climater implementation has been removed as it drains battery. Re-check at a later time.

// var _ api.VehiclePosition = (*Tesla)(nil)

// // Position implements the api.VehiclePosition interface
// func (v *Tesla) Position() (float64, float64, error) {
// 	res, err := v.driveStateG()
// 	if err == nil {
// 		return res.Latitude, res.Longitude, nil
// 	}

// 	return 0, 0, err
// }

// var _ api.SocLimiter = (*Tesla)(nil)

// // TargetSoC implements the api.SocLimiter interface
// func (v *Tesla) TargetSoC() (float64, error) {
// 	res, err := v.chargeStateG()
// 	if err == nil {
// 		return float64(res.ChargeLimitSoc), nil
// 	}

// 	return 0, err
// }

// var _ api.VehicleChargeController = (*Tesla)(nil)

// // StartCharge implements the api.VehicleChargeController interface
// func (v *Tesla) StartCharge() error {
// 	err := v.vehicle.StartCharging()

// 	if err != nil && err.Error() == "408 Request Timeout" {
// 		if _, err := v.vehicle.Wakeup(); err != nil {
// 			return err
// 		}

// 		timer := time.NewTimer(90 * time.Second)

// 		for {
// 			select {
// 			case <-timer.C:
// 				return api.ErrTimeout
// 			default:
// 				time.Sleep(2 * time.Second)
// 				if err := v.vehicle.StartCharging(); err == nil || err.Error() != "408 Request Timeout" {
// 					return err
// 				}
// 			}
// 		}
// 	}

// 	return err
// }

// // StopCharge implements the api.VehicleChargeController interface
// func (v *Tesla) StopCharge() error {
// 	err := v.vehicle.StopCharging()

// 	// ignore sleeping vehicle
// 	if err != nil && err.Error() == "408 Request Timeout" {
// 		err = nil
// 	}

// 	return err
// }
