package charger

import (
	"encoding/json"
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

const (
	apiSettings     apiFunction = "settings"
	apiMeasurements apiFunction = "measurements"
)

// NRGResponse is the API response if status not OK
type NRGResponse struct {
	Message string
}

// NRGMeasurements is the /api/measurements response
type NRGMeasurements struct {
	ChargingEnergy        float64
	ChargingEnergyOverAll float64
	ChargingPower         float64
	ChargingPowerPhase    [3]float64
	ChargingCurrentPhase  [3]float64
	Frequency             float64
}

// NRGSettings is the /api/settings request/response
type NRGSettings struct {
	Info   NRGInfo `json:"omitempty"`
	Values NRGValues
}

// NRGInfo is NRGSettings.Info
type NRGInfo struct {
	Connected bool `json:"omitempty"`
}

// NRGValues is NRGSettings.Values
type NRGValues struct {
	ChargingStatus  NRGChargingStatus
	ChargingCurrent NRGChargingCurrent
	DeviceMetadata  NRGDeviceMetadata
}

// NRGChargingStatus is NRGSettings.Values.ChargingStatus
type NRGChargingStatus struct {
	Charging *bool `json:"omitempty"` // use pointer to allow omitting false
}

// NRGChargingCurrent is NRGSettings.Values.ChargingCurrent
type NRGChargingCurrent struct {
	Value float64 `json:"omitempty"`
}

// NRGDeviceMetadata is NRGSettings.Values.DeviceMetadata
type NRGDeviceMetadata struct {
	Password string
}

// NRGKickConnect charger implementation
type NRGKickConnect struct {
	*util.HTTPHelper
	IP         string
	MacAddress string
	Password   string
}

// NewNRGKickConnectFromConfig creates a NRGKickConnect charger from generic config
func NewNRGKickConnectFromConfig(log *util.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ IP, MacAddress, Password string }{}
	util.DecodeOther(log, other, &cc)

	return NewNRGKickConnect(cc.IP, cc.MacAddress, cc.Password)
}

// NewNRGKickConnect creates NRGKickConnect charger
func NewNRGKickConnect(IP, MacAddress, Password string) *NRGKickConnect {
	nrg := &NRGKickConnect{
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("nrgc")),
		IP:         IP,
		MacAddress: MacAddress,
		Password:   Password,
	}

	nrg.HTTPHelper.Log.WARN.Println("-- experimental --")

	return nrg
}

func (nrg *NRGKickConnect) apiURL(api apiFunction) string {
	return fmt.Sprintf("%s/api/%s/%s", nrg.IP, api, nrg.MacAddress)
}

func (nrg *NRGKickConnect) getJSON(url string, result interface{}) error {
	b, err := nrg.GetJSON(url, result)
	if err != nil && len(b) > 0 {
		var error NRGResponse
		if err := json.Unmarshal(b, &error); err != nil {
			return err
		}

		return fmt.Errorf("response: %s", error.Message)
	}

	return err
}

func (nrg *NRGKickConnect) putJSON(url string, request interface{}) error {
	b, err := nrg.PutJSON(url, request)
	if err != nil && len(b) == 0 {
		return err
	}

	var error NRGResponse
	if err := json.Unmarshal(b, &error); err != nil {
		return err
	}

	return fmt.Errorf("response: %s", error.Message)
}

// Status implements the Charger.Status interface
func (nrg *NRGKickConnect) Status() (api.ChargeStatus, error) {
	return api.StatusC, nil
}

// Enabled implements the Charger.Enabled interface
func (nrg *NRGKickConnect) Enabled() (bool, error) {
	var settings NRGSettings
	err := nrg.getJSON(nrg.apiURL(apiSettings), settings)

	return *settings.Values.ChargingStatus.Charging, err
}

// Enable implements the Charger.Enable interface
func (nrg *NRGKickConnect) Enable(enable bool) error {
	settings := NRGSettings{}
	settings.Values.DeviceMetadata.Password = nrg.Password
	settings.Values.ChargingStatus.Charging = &enable

	return nrg.putJSON(nrg.apiURL(apiSettings), settings)
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (nrg *NRGKickConnect) MaxCurrent(current int64) error {
	settings := NRGSettings{}
	settings.Values.DeviceMetadata.Password = nrg.Password
	settings.Values.ChargingCurrent.Value = float64(current)

	return nrg.putJSON(nrg.apiURL(apiSettings), settings)
}

// CurrentPower implements the Meter interface
func (nrg *NRGKickConnect) CurrentPower() (float64, error) {
	var measurements NRGMeasurements
	err := nrg.getJSON(nrg.apiURL(apiMeasurements), measurements)

	return 1000 * measurements.ChargingPower, err
}

// TotalEnergy implements the MeterEnergy interface
func (nrg *NRGKickConnect) TotalEnergy() (float64, error) {
	var measurements NRGMeasurements
	err := nrg.getJSON(nrg.apiURL(apiMeasurements), measurements)

	return measurements.ChargingEnergyOverAll, err
}

// Currents implements the MeterCurrent interface
func (nrg *NRGKickConnect) Currents() (float64, float64, float64, error) {
	var measurements NRGMeasurements
	err := nrg.getJSON(nrg.apiURL(apiMeasurements), measurements)

	if len(measurements.ChargingCurrentPhase) != 3 {
		return 0, 0, 0, fmt.Errorf("unexpected response: %v", measurements)
	}

	return measurements.ChargingCurrentPhase[0],
		measurements.ChargingCurrentPhase[1],
		measurements.ChargingCurrentPhase[2],
		err
}

// ChargedEnergy implements the ChargeRater interface
// NOTE: apparently shows energy of a stopped charging session, hence substituted by TotalEnergy
// func (nrg *NRGKickConnect) ChargedEnergy() (float64, error) {
// 	var measurements NRGMeasurements
// 	err := nrg.getJSON(nrg.apiURL(apiMeasurements), measurements)
// 	return measurements.ChargingEnergy, err
// }
