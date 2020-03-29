package charger

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andig/evcc/api"
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

// NRGSettings is the /api/setings request/response
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
	log        *api.Logger
	IP         string
	MacAddress string
	Password   string
}

// NewNRGKickFromConfig creates a NRGKick charger from generic config
func NewNRGKickFromConfig(log *api.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ IP, MacAddress, Password string }{}
	api.DecodeOther(log, other, &cc)

	return NewNRGKick(cc.IP, cc.MacAddress, cc.Password)
}

// NewNRGKick creates NRGKick charger
func NewNRGKick(IP, MacAddress, Password string) *NRGKick {
	nrg := &NRGKick{
		IP:         IP,
		MacAddress: MacAddress,
		Password:   Password,
		log:        api.NewLogger("kick"),
	}

	nrg.log.WARN.Println("-- experimental --")

	return nrg
}

func (nrg *NRGKick) apiURL(api apiFunction) string {
	return fmt.Sprintf("%s/api/%s/%s", nrg.IP, api, nrg.MacAddress)
}

func (nrg *NRGKick) getJSON(url string, result interface{}) error {
	resp, body, err := getJSON(url, result)
	nrg.log.TRACE.Printf("GET %s: %s", url, string(body))

	if err != nil && len(body) == 0 {
		return err
	}

	var error NRGResponse
	_ = json.Unmarshal(body, &error)

	return fmt.Errorf("api %d: %s", resp.StatusCode, error.Message)
}

func (nrg *NRGKick) putJSON(url string, request interface{}) error {
	resp, body, err := putJSON(url, request)
	nrg.log.TRACE.Printf("PUT %v: %s", resp, string(body))

	if err != nil && len(body) == 0 {
		return err
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	var error NRGResponse
	_ = json.Unmarshal(body, &error)

	return fmt.Errorf("api %d: %s", resp.StatusCode, error.Message)
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
