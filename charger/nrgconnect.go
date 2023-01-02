package charger

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// https://www.nrgkick.com/wp-content/uploads/2019/08/20190814_API-Dokumentation_04.pdf

const (
	nrgSettings     = "settings"
	nrgMeasurements = "measurements"
)

// NRGMeasurements is the /api/measurements response
type NRGMeasurements struct {
	Message               string `json:"omitempty"` // api message if not ok
	ChargingEnergy        float64
	ChargingEnergyOverAll float64
	ChargingPower         float64
	ChargingPowerPhase    [3]float64
	ChargingCurrentPhase  [3]float64
	Frequency             float64
}

// NRGSettings is the /api/settings request/response
type NRGSettings struct {
	Message string  `json:",omitempty"` // api message if not ok
	Info    NRGInfo `json:",omitempty"`
	Values  NRGValues
}

// NRGInfo is NRGSettings.Info
type NRGInfo struct {
	Connected bool `json:",omitempty"`
}

// NRGValues is NRGSettings.Values
type NRGValues struct {
	ChargingStatus  NRGChargingStatus
	ChargingCurrent NRGChargingCurrent
	DeviceMetadata  NRGDeviceMetadata
}

// NRGChargingStatus is NRGSettings.Values.ChargingStatus
type NRGChargingStatus struct {
	Charging *bool `json:",omitempty"` // use pointer to allow omitting false
}

// NRGChargingCurrent is NRGSettings.Values.ChargingCurrent
type NRGChargingCurrent struct {
	Value float64 `json:",omitempty"`
}

// NRGDeviceMetadata is NRGSettings.Values.DeviceMetadata
type NRGDeviceMetadata struct {
	Password string
}

// NRGKickConnect charger implementation
type NRGKickConnect struct {
	*request.Helper
	uri      string
	mac      string
	password string
}

func init() {
	registry.Add("nrgkick-connect", NewNRGKickConnectFromConfig)
}

// NewNRGKickConnectFromConfig creates a NRGKickConnect charger from generic config
func NewNRGKickConnectFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		URI, Mac, Password string
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewNRGKickConnect(cc.URI, cc.Mac, cc.Password)
}

// NewNRGKickConnect creates NRGKickConnect charger
func NewNRGKickConnect(uri, mac, password string) (*NRGKickConnect, error) {
	nrg := &NRGKickConnect{
		Helper:   request.NewHelper(util.NewLogger("nrgconn")),
		uri:      util.DefaultScheme(uri, "http"),
		mac:      mac,
		password: password,
	}

	return nrg, nil
}

func (nrg *NRGKickConnect) apiURL(api string) string {
	return fmt.Sprintf("%s/api/%s/%s", nrg.uri, api, nrg.mac)
}

func (nrg *NRGKickConnect) putJSON(url string, data interface{}) error {
	req, err := request.New(http.MethodPut, url, request.MarshalJSON(data))

	if err == nil {
		var resp struct {
			Message string
		}

		if err = nrg.DoJSON(req, &resp); err != nil {
			if resp.Message != "" {
				return fmt.Errorf("response: %s", resp.Message)
			}
		}
	}

	return err
}

// Status implements the api.Charger interface
func (nrg *NRGKickConnect) Status() (api.ChargeStatus, error) {
	return api.StatusC, nil
}

// Enabled implements the api.Charger interface
func (nrg *NRGKickConnect) Enabled() (bool, error) {
	var res NRGSettings
	err := nrg.GetJSON(nrg.apiURL(nrgSettings), &res)
	if err != nil {
		if res.Message != "" {
			err = errors.New(res.Message)
		}

		return false, err
	}

	return *res.Values.ChargingStatus.Charging, nil
}

// Enable implements the api.Charger interface
func (nrg *NRGKickConnect) Enable(enable bool) error {
	settings := NRGSettings{}
	settings.Values.DeviceMetadata.Password = nrg.password
	settings.Values.ChargingStatus.Charging = &enable

	return nrg.putJSON(nrg.apiURL(nrgSettings), settings)
}

// MaxCurrent implements the api.Charger interface
func (nrg *NRGKickConnect) MaxCurrent(current int64) error {
	settings := NRGSettings{}
	settings.Values.DeviceMetadata.Password = nrg.password
	settings.Values.ChargingCurrent.Value = float64(current)

	return nrg.putJSON(nrg.apiURL(nrgSettings), settings)
}

var _ api.Meter = (*NRGKickConnect)(nil)

// CurrentPower implements the api.Meter interface
func (nrg *NRGKickConnect) CurrentPower() (float64, error) {
	var res NRGMeasurements
	err := nrg.GetJSON(nrg.apiURL(nrgMeasurements), &res)
	if err != nil && res.Message != "" {
		err = errors.New(res.Message)
	}

	return 1000 * res.ChargingPower, err
}

var _ api.MeterEnergy = (*NRGKickConnect)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (nrg *NRGKickConnect) TotalEnergy() (float64, error) {
	var res NRGMeasurements
	err := nrg.GetJSON(nrg.apiURL(nrgMeasurements), &res)
	if err != nil && res.Message != "" {
		err = errors.New(res.Message)
	}

	return res.ChargingEnergyOverAll, err
}

var _ api.PhaseCurrents = (*NRGKickConnect)(nil)

// Currents implements the api.PhaseCurrents interface
func (nrg *NRGKickConnect) Currents() (float64, float64, float64, error) {
	var res NRGMeasurements
	err := nrg.GetJSON(nrg.apiURL(nrgMeasurements), &res)
	if err != nil && res.Message != "" {
		err = errors.New(res.Message)
	}

	if len(res.ChargingCurrentPhase) != 3 {
		return 0, 0, 0, fmt.Errorf("unexpected response: %v", res)
	}

	return res.ChargingCurrentPhase[0],
		res.ChargingCurrentPhase[1],
		res.ChargingCurrentPhase[2],
		err
}

// ChargedEnergy implements the ChargeRater interface
// NOTE: apparently shows energy of a stopped charging session, hence substituted by TotalEnergy
// func (nrg *NRGKickConnect) ChargedEnergy() (float64, error) {
// 	var res NRGMeasurements
// 	err := nrg.GetJSON(nrg.apiURL(nrgMeasurements), &res)
// 	return res.ChargingEnergy, err
// }
