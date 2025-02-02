package charger

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/nrg/connect"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// https://www.nrgkick.com/wp-content/uploads/2019/08/20190814_API-Dokumentation_04.pdf

// NRGKickConnect charger implementation
type NRGKickConnect struct {
	*request.Helper
	uri           string
	mac           string
	password      string
	enabled       bool
	settingsG     provider.Cacheable[connect.Settings]
	measurementsG provider.Cacheable[connect.Measurements]
}

func init() {
	registry.Add("nrgkick-connect", NewNRGKickConnectFromConfig)
}

// NewNRGKickConnectFromConfig creates a NRGKickConnect charger from generic config
func NewNRGKickConnectFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI, Mac, Password string
		Cache              time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewNRGKickConnect(cc.URI, cc.Mac, cc.Password, cc.Cache)
}

// NewNRGKickConnect creates NRGKickConnect charger
func NewNRGKickConnect(uri, mac, password string, cache time.Duration) (*NRGKickConnect, error) {
	nrg := &NRGKickConnect{
		Helper:   request.NewHelper(util.NewLogger("nrgconn")),
		uri:      util.DefaultScheme(uri, "http"),
		mac:      mac,
		password: password,
	}

	nrg.settingsG = provider.ResettableCached(func() (connect.Settings, error) {
		var res connect.Settings

		err := nrg.GetJSON(nrg.apiURL(connect.SettingsPath), &res)
		if err != nil && res.Message != "" {
			err = errors.New(res.Message)
		}

		return res, err
	}, cache)

	nrg.measurementsG = provider.ResettableCached(func() (connect.Measurements, error) {
		var res connect.Measurements

		err := nrg.GetJSON(nrg.apiURL(connect.MeasurementsPath), &res)
		if err != nil && res.Message != "" {
			err = errors.New(res.Message)
		}

		return res, err
	}, cache)

	return nrg, nil
}

func (nrg *NRGKickConnect) apiURL(api string) string {
	return fmt.Sprintf("%s/api/%s/%s", nrg.uri, api, nrg.mac)
}

func (nrg *NRGKickConnect) putJSON(url string, data interface{}) error {
	req, err := request.New(http.MethodPut, url, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res struct {
		Message string
	}

	if err := nrg.DoJSON(req, &res); err != nil {
		switch {
		case res.Message != "":
			return errors.New(res.Message)
		case err != io.EOF:
			return err
		}
	}

	nrg.settingsG.Reset()
	nrg.measurementsG.Reset()

	return nil
}

// Status implements the api.Charger interface
func (nrg *NRGKickConnect) Status() (api.ChargeStatus, error) {
	res, err := nrg.settingsG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	if res.Values.ChargingStatus == nil {
		return api.StatusNone, errors.New("unknown status")
	}

	if res.Values.ChargingStatus.Charging {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (nrg *NRGKickConnect) Enabled() (bool, error) {
	return nrg.enabled, nil
}

// Enable implements the api.Charger interface
func (nrg *NRGKickConnect) Enable(enable bool) error {
	data := connect.Settings{
		Values: connect.Values{
			ChargingStatus: &connect.ChargingStatus{
				Charging: enable,
			},
			DeviceMetadata: connect.DeviceMetadata{
				Password: nrg.password,
			},
		},
	}

	err := nrg.putJSON(nrg.apiURL(connect.SettingsPath), data)
	if err == nil {
		nrg.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (nrg *NRGKickConnect) MaxCurrent(current int64) error {
	data := connect.Settings{
		Values: connect.Values{
			ChargingStatus: &connect.ChargingStatus{
				Charging: nrg.enabled,
			},
			ChargingCurrent: &connect.ChargingCurrent{
				Value: float64(current),
			},
			DeviceMetadata: connect.DeviceMetadata{
				Password: nrg.password,
			},
		},
	}

	return nrg.putJSON(nrg.apiURL(connect.SettingsPath), data)
}

var _ api.Meter = (*NRGKickConnect)(nil)

// CurrentPower implements the api.Meter interface
func (nrg *NRGKickConnect) CurrentPower() (float64, error) {
	res, err := nrg.measurementsG.Get()
	if err != nil {
		return 0, err
	}

	return res.ChargingPower * 1e3, err
}

var _ api.MeterEnergy = (*NRGKickConnect)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (nrg *NRGKickConnect) TotalEnergy() (float64, error) {
	res, err := nrg.measurementsG.Get()
	if err != nil {
		return 0, err
	}

	return res.ChargingEnergyOverAll, err
}

var _ api.PhaseCurrents = (*NRGKickConnect)(nil)

// Currents implements the api.PhaseCurrents interface
func (nrg *NRGKickConnect) Currents() (float64, float64, float64, error) {
	res, err := nrg.measurementsG.Get()
	if err != nil {
		return 0, 0, 0, err
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
// 	var res connect.Measurements
// 	err := nrg.GetJSON(nrg.apiURL(connect.MeasurementsPath), &res)
// 	return res.ChargingEnergy, err
// }
