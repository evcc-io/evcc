package leapmotor

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// API makes authenticated calls to the Leapmotor cloud API.
type API struct {
	log      *util.Logger
	identity *Identity
}

// NewAPI creates a new API client backed by the given Identity.
func NewAPI(log *util.Logger, identity *Identity) *API {
	return &API{log: log, identity: identity}
}

// do sends an authenticated POST to path, retrying once on token expiry.
func (a *API) do(path, vin string, body string) ([]byte, error) {
	doOnce := func() ([]byte, error) {
		client, token, userID, deviceID, signKey := a.identity.Session()
		if client == nil {
			return nil, fmt.Errorf("not authenticated")
		}
		headers := buildSignedHeaders(signKey, deviceID, vin, defaultLang, userID, token, nil)
		return apiPost(client, BaseURL+path, headers, body)
	}

	respBody, err := doOnce()
	if err != nil {
		return nil, err
	}

	// Retry once on token-related API errors.
	var env apiEnvelope[json.RawMessage]
	if json.Unmarshal(respBody, &env) == nil && env.Code != 0 &&
		strings.Contains(strings.ToLower(env.Message), "token") {
		a.log.DEBUG.Printf("token error (%d: %s), refreshing", env.Code, env.Message)
		if err := a.identity.Refresh(); err != nil {
			return nil, err
		}
		return doOnce()
	}

	return respBody, nil
}

// Vehicles returns all owned and shared vehicles on the account.
func (a *API) Vehicles() ([]Vehicle, error) {
	body, err := a.do("/carownerservice/oversea/vehicle/v1/list", "", "")
	if err != nil {
		return nil, err
	}
	type listData struct {
		Bindcars   []Vehicle `json:"bindcars"`
		Sharedcars []Vehicle `json:"sharedcars"`
	}
	data, err := parseEnvelope[listData](body)
	if err != nil {
		return nil, err
	}
	all := append(data.Bindcars, data.Sharedcars...)
	valid := slices.DeleteFunc(all, func(v Vehicle) bool {
		return v.VIN == ""
	})
	return valid, nil
}

// Status fetches the current status for the given VIN and car type.
func (a *API) Status(vin, carType string) (StatusData, error) {
	path := "/carownerservice/oversea/vehicle/v1/status/get/" + strings.ToLower(carType)
	reqBody := "vin=" + url.QueryEscape(vin)
	body, err := a.do(path, vin, reqBody)
	if err != nil {
		return StatusData{}, err
	}
	return parseEnvelope[StatusData](body)
}

// Provider implements the evcc vehicle interfaces using a cached status call.
type Provider struct {
	status util.Cacheable[StatusData]
}

// NewProvider creates a Provider that caches the status for the given VIN.
func NewProvider(api *API, vin, carType string, cache time.Duration) *Provider {
	return &Provider{
		status: util.ResettableCached(func() (StatusData, error) {
			return api.Status(vin, carType)
		}, cache),
	}
}

var _ api.Battery = (*Provider)(nil)

// Soc implements api.Battery.
func (p *Provider) Soc() (float64, error) {
	res, err := p.status.Get()
	if err != nil {
		return 0, err
	}
	if res.Soc == nil {
		return 0, api.ErrMustRetry
	}
	return float64(*res.Soc), nil
}

var _ api.ChargeState = (*Provider)(nil)

// Status implements api.ChargeState.
// ChargeState 0 = not connected, 1/2 = AC/DC connected.
// Charging is detected by negative battery power with remaining charge time.
func (p *Provider) Status() (api.ChargeStatus, error) {
	res, err := p.status.Get()
	if err != nil {
		return api.StatusNone, err
	}
	if res.ChargeState == nil || *res.ChargeState == 0 {
		return api.StatusA, nil
	}
	// Plugged in: determine if actively charging.
	if res.BatteryVoltage != nil && res.BatteryCurrent != nil && res.ChargeRemainTime != nil {
		power := (*res.BatteryVoltage) * (*res.BatteryCurrent) / 1000 // kW
		if power < 0 && *res.ChargeRemainTime > 0 {
			return api.StatusC, nil
		}
	}
	return api.StatusB, nil
}

var _ api.VehicleRange = (*Provider)(nil)

// Range implements api.VehicleRange.
func (p *Provider) Range() (int64, error) {
	res, err := p.status.Get()
	if err != nil {
		return 0, err
	}
	if res.ExpectedMileage == nil {
		return 0, api.ErrMustRetry
	}
	return int64(*res.ExpectedMileage), nil
}

var _ api.VehicleOdometer = (*Provider)(nil)

// Odometer implements api.VehicleOdometer.
func (p *Provider) Odometer() (float64, error) {
	res, err := p.status.Get()
	if err != nil {
		return 0, err
	}
	if res.TotalMileage == nil {
		return 0, api.ErrMustRetry
	}
	return float64(*res.TotalMileage), nil
}

var _ api.VehicleFinishTimer = (*Provider)(nil)

// FinishTime implements api.VehicleFinishTimer.
func (p *Provider) FinishTime() (time.Time, error) {
	res, err := p.status.Get()
	if err != nil {
		return time.Time{}, err
	}
	if res.ChargeRemainTime == nil || *res.ChargeRemainTime <= 0 {
		return time.Time{}, api.ErrMustRetry
	}
	return time.Now().Add(time.Duration(*res.ChargeRemainTime) * time.Minute), nil
}

var _ api.VehicleClimater = (*Provider)(nil)

// Climater implements api.VehicleClimater.
func (p *Provider) Climater() (bool, error) {
	res, err := p.status.Get()
	if err != nil {
		return false, err
	}
	if res.AcSwitch == nil {
		return false, api.ErrMustRetry
	}
	return *res.AcSwitch, nil
}

var _ api.VehiclePosition = (*Provider)(nil)

// Position implements api.VehiclePosition.
func (p *Provider) Position() (float64, float64, error) {
	res, err := p.status.Get()
	if err != nil {
		return 0, 0, err
	}
	if res.Latitude == nil || res.Longitude == nil {
		return 0, 0, api.ErrMustRetry
	}
	return *res.Latitude, *res.Longitude, nil
}

var _ api.SocLimiter = (*Provider)(nil)

// GetLimitSoc implements api.SocLimiter.
func (p *Provider) GetLimitSoc() (int64, error) {
	res, err := p.status.Get()
	if err != nil {
		return 0, err
	}
	if res.ChargeSocSetting == nil {
		return 0, api.ErrMustRetry
	}
	return int64(*res.ChargeSocSetting), nil
}
