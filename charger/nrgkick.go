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
	ChargingEnergy float64
	ChargingPower  float64
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

// NRGKick charger implementation
type NRGKick struct {
	*util.HTTPHelper
	IP         string
	MacAddress string
	Password   string
}

// NewNRGKickFromConfig creates a NRGKick charger from generic config
func NewNRGKickFromConfig(log *util.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ IP, MacAddress, Password string }{}
	util.DecodeOther(log, other, &cc)

	return NewNRGKick(cc.IP, cc.MacAddress, cc.Password)
}

// NewNRGKick creates NRGKick charger
func NewNRGKick(IP, MacAddress, Password string) *NRGKick {
	nrg := &NRGKick{
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("kick")),
		IP:         IP,
		MacAddress: MacAddress,
		Password:   Password,
	}

	nrg.HTTPHelper.Log.WARN.Println("-- experimental --")

	return nrg
}

func (nrg *NRGKick) apiURL(api apiFunction) string {
	return fmt.Sprintf("%s/api/%s/%s", nrg.IP, api, nrg.MacAddress)
}

func (nrg *NRGKick) getJSON(url string, result interface{}) error {
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

func (nrg *NRGKick) putJSON(url string, request interface{}) error {
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
func (nrg *NRGKick) Status() (api.ChargeStatus, error) {
	return api.StatusC, nil
}

// Enabled implements the Charger.Enabled interface
func (nrg *NRGKick) Enabled() (bool, error) {
	var settings NRGSettings
	err := nrg.getJSON(nrg.apiURL(apiSettings), settings)

	return *settings.Values.ChargingStatus.Charging, err
}

// Enable implements the Charger.Enable interface
func (nrg *NRGKick) Enable(enable bool) error {
	settings := NRGSettings{}
	settings.Values.DeviceMetadata.Password = nrg.Password
	settings.Values.ChargingStatus.Charging = &enable

	return nrg.putJSON(nrg.apiURL(apiSettings), settings)
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (nrg *NRGKick) MaxCurrent(current int64) error {
	settings := NRGSettings{}
	settings.Values.DeviceMetadata.Password = nrg.Password
	settings.Values.ChargingCurrent.Value = float64(current)

	return nrg.putJSON(nrg.apiURL(apiSettings), settings)
}

// CurrentPower implements the Meter interface.
func (nrg *NRGKick) CurrentPower() (float64, error) {
	var measurements NRGMeasurements
	err := nrg.getJSON(nrg.apiURL(apiMeasurements), measurements)

	return 1000 * measurements.ChargingPower, err
}

// ChargedEnergy implements the ChargeRater interface.
func (nrg *NRGKick) ChargedEnergy() (float64, error) {
	var measurements NRGMeasurements
	err := nrg.getJSON(nrg.apiURL(apiMeasurements), measurements)

	return measurements.ChargingEnergy, err
}
