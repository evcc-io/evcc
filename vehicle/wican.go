package vehicle

import (
	"fmt"
	"net"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// WiCAN is an api.Vehicle implementation for the WiCAN Pro API
type WiCAN struct {
	*embed
	helper *request.Helper
	host   string
	dataG  func() (WiCANResponse, error)
}

type WiCANResponse struct {
	Soc      *float64 `json:"SOC"`
	SocD     *float64 `json:"SOC_D"`
	Range    *float64 `json:"RANGE"`
	Odometer *float64 `json:"ODOMETER"`
	Charging *int64   `json:"CHARGING"`
	Climater *int64   `json:"CLIMATER"`
}

func init() {
	registry.Add("wican", NewWiCANFromConfig)
}

// NewWiCANFromConfig creates a new vehicle
func NewWiCANFromConfig(other map[string]any) (api.Vehicle, error) {
	cc := struct {
		embed `mapstructure:",squash"`
		Host  string
		Cache time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("wican")

	v := &WiCAN{
		embed:  &cc.embed,
		host:   cc.Host,
		helper: request.NewHelper(log),
	}

	v.dataG = util.Cached(v.fetchData, cc.Cache)

	return v, nil
}

// fetchData fetches the live data snapshot from the WiCAN Pro API
func (v *WiCAN) fetchData() (WiCANResponse, error) {
	var res WiCANResponse

	host := net.JoinHostPort(v.host, "80")
	uri := fmt.Sprintf("http://%s/api/autopid/data", host)

	err := v.helper.GetJSON(uri, &res)
	if err != nil {
		return res, fmt.Errorf("WiCAN API error: %w", err)
	}

	return res, nil
}

// Soc implements the api.Vehicle interface
func (v *WiCAN) Soc() (float64, error) {
	res, err := v.dataG()

	if err != nil {
		return 0, err
	}

	// use "display SoC", if value exists
	if res.SocD != nil {
		return *res.SocD, err
	}

	// otherwise use real SoC
	if res.Soc != nil {
		return *res.Soc, err
	}

	return 0, fmt.Errorf("Missing SOC_D or SOC value in WiCAN API response")
}

// Range implements the api.VehicleRange interface
func (v *WiCAN) Range() (int64, error) {
	res, err := v.dataG()

	if err != nil {
		return 0, err
	}

	if res.Range == nil {
		return 0, api.ErrNotAvailable
	}

	return int64(*res.Range), nil
}

// Odometer implements the api.VehicleOdometer interface
func (v *WiCAN) Odometer() (float64, error) {
	res, err := v.dataG()

	if err != nil {
		return 0, err
	}

	if res.Odometer == nil {
		return 0, api.ErrNotAvailable
	}

	return *res.Odometer, nil
}

// Status implements the api.ChargeState interface
func (v *WiCAN) Status() (api.ChargeStatus, error) {
	res, err := v.dataG()

	if err != nil {
		return api.StatusNone, err
	}

	if res.Charging == nil {
		return api.StatusNone, api.ErrNotAvailable
	}

	if *res.Charging == 1 {
		return api.StatusC, nil // charging
	}

	return api.StatusNone, nil // unknown (A or B)
}

// Climater implements the  api.VehicleClimater interface
func (v *WiCAN) Climater() (bool, error) {
	res, err := v.dataG()

	if err != nil {
		return false, err
	}

	if res.Climater == nil {
		return false, api.ErrNotAvailable
	}

	return *res.Climater == 1, nil
}
