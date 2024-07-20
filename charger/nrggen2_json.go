package charger

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/nrg/gen2"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// https://www.nrgkick.com/wp-content/uploads/2024/07/local_api_docu_simulate.html

// NRGKickGen2 charger implementation
type NRGKickGen2 struct {
	*request.Helper
	uri      string
	password string
	enabled  bool
	controlG provider.Cacheable[gen2.Control]
	valuesG  provider.Cacheable[gen2.Values]
	infoG    provider.Cacheable[gen2.Info]
}

func init() {
	registry.Add("nrgkick-gen2", NewNRGKickGen2FromConfig)
}

// NewNRGKickGen2FromConfig creates a NRGKickGen2 charger from generic config
func NewNRGKickGen2FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI, User, Password string
		Cache               time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewNRGKickGen2(cc.URI, cc.User, cc.Password, cc.Cache)
}

// NewNRGKickGen2 creates NRGKickGen2 charger
func NewNRGKickGen2(uri, user, password string, cache time.Duration) (*NRGKickGen2, error) {
	basicAuth := transport.BasicAuthHeader(user, password)
	log := util.NewLogger("nrggen2").Redact(user, password, basicAuth)

	nrg := &NRGKickGen2{
		Helper:   request.NewHelper(log),
		uri:      util.DefaultScheme(uri, "http"),
		password: password,
	}

	if user != "" && password != "" {
		nrg.Client.Transport = transport.BasicAuth(user, password, nrg.Client.Transport)
	}

	nrg.controlG = provider.ResettableCached(func() (gen2.Control, error) {
		var res gen2.Control

		err := nrg.GetJSON(nrg.apiURL(gen2.ControlPath), &res)
		if err != nil && res.Response != "" {
			err = errors.New(res.Response)
		}

		return res, err
	}, cache)

	nrg.valuesG = provider.ResettableCached(func() (gen2.Values, error) {
		var res gen2.Values

		err := nrg.GetJSON(nrg.apiURL(gen2.ValuesPath), &res)

		return res, err
	}, cache)

	nrg.infoG = provider.ResettableCached(func() (gen2.Info, error) {
		var res gen2.Info

		err := nrg.GetJSON(nrg.apiURL(gen2.InfoPath), &res)

		return res, err
	}, cache)

	return nrg, nil
}

func (nrg *NRGKickGen2) apiURL(api string) string {
	return fmt.Sprintf("%s/%s", nrg.uri, api)
}

func (nrg *NRGKickGen2) updateControl(control gen2.Control, withPhaseSwitch bool) error {
	// TODO: i am not sure if we can set current_set with decimals over the query parameter, so just in case i limit it to the integer value
	uriWithQueryParams := fmt.Sprintf(
		"%s?current_set=%.0f&charge_pause=%d&energy_limit=%d",
		nrg.apiURL(gen2.ControlPath),
		control.CurrentSet,
		control.ChargePause,
		control.EnergyLimit)

	if withPhaseSwitch {
		uriWithQueryParams = fmt.Sprintf("%s&phase_count=%d", uriWithQueryParams, control.PhaseCount)
	}

	req, err := request.New(http.MethodGet, uriWithQueryParams, nil, request.AcceptJSON)
	if err != nil {
		return err
	}

	var res gen2.Control

	if err := nrg.DoJSON(req, &res); err != nil {
		switch {
		case res.Response != "":
			return errors.New(res.Response)
		case err != io.EOF:
			return err
		}
	}

	nrg.controlG.Reset()
	nrg.valuesG.Reset()

	return nil
}

// Status implements the api.Charger interface
func (nrg *NRGKickGen2) Status() (api.ChargeStatus, error) {
	res, err := nrg.valuesG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	switch res.General.Status {
	case "UNKNOWN":
		return api.StatusNone, nil
	case "STANDBY":
		return api.StatusA, nil
	case "CONNECTED":
		return api.StatusB, nil
	case "CHARGING":
		return api.StatusC, nil
	case "ERROR":
		switch res.General.ErrorCode {
		case "CP_SIGNAL_VOLTAGE_ERROR":
			fallthrough
		case "CP_SIGNAL_IMPERMISSIBLE":
			return api.StatusE, fmt.Errorf("%s", res.General.ErrorCode)
		default:
			return api.StatusF, fmt.Errorf("%s", res.General.ErrorCode)
		}
	case "WAKEUP":
		return api.StatusNone, nil
	default:
		return api.StatusNone, fmt.Errorf("unhandled status type")
	}
}

// Enabled implements the api.Charger interface
func (nrg *NRGKickGen2) Enabled() (bool, error) {
	return nrg.enabled, nil
}

// Enable implements the api.Charger interface
func (nrg *NRGKickGen2) Enable(enable bool) error {
	res, err := nrg.controlG.Get()

	if err != nil {
		return err
	}

	if enable {
		res.ChargePause = 0
	} else {
		res.ChargePause = 1
	}

	err = nrg.updateControl(res, false)

	if err == nil {
		nrg.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (nrg *NRGKickGen2) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("allowed range: 6.0 - rated_current (16.0A / 32.0A)")
	}

	res, err := nrg.controlG.Get()

	if err != nil {
		return err
	}

	res.CurrentSet = float64(current)

	return nrg.updateControl(res, false)
}

func (nrg *NRGKickGen2) GetMaxCurrent() (float64, error) {
	res, err := nrg.controlG.Get()

	if err != nil {
		return 0, err
	}

	return res.CurrentSet, nil
}

var _ api.Meter = (*NRGKickGen2)(nil)

// CurrentPower implements the api.Meter interface
func (nrg *NRGKickGen2) CurrentPower() (float64, error) {
	res, err := nrg.valuesG.Get()

	if err != nil {
		return 0, err
	}

	return res.Powerflow.TotalActivePower, err
}

var _ api.MeterEnergy = (*NRGKickGen2)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (nrg *NRGKickGen2) TotalEnergy() (float64, error) {
	res, err := nrg.valuesG.Get()
	if err != nil {
		return 0, err
	}

	return float64(res.Energy.TotalChargedEnergy) * 1e-3, err
}

var _ api.PhaseCurrents = (*NRGKickGen2)(nil)

// Currents implements the api.PhaseCurrents interface
func (nrg *NRGKickGen2) Currents() (float64, float64, float64, error) {
	res, err := nrg.valuesG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.Powerflow.L1.Current,
		res.Powerflow.L1.Current,
		res.Powerflow.L1.Current,
		err
}

var _ api.PhaseVoltages = (*NRGKickGen2)(nil)

// Currents implements the api.PhaseVoltages interface
func (nrg *NRGKickGen2) Voltages() (float64, float64, float64, error) {
	res, err := nrg.valuesG.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	return res.Powerflow.L1.Voltage,
		res.Powerflow.L1.Voltage,
		res.Powerflow.L1.Voltage,
		err
}

var _ api.ChargeRater = (*NRGKickGen2)(nil)

func (nrg *NRGKickGen2) ChargedEnergy() (float64, error) {
	res, err := nrg.valuesG.Get()
	if err != nil {
		return 0, err
	}

	return float64(res.Energy.ChargedEnergy) * 1e-3, err
}

var _ api.PhaseSwitcher = (*NRGKickGen2)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (nrg *NRGKickGen2) Phases1p3p(phases int) error {
	res, err := nrg.controlG.Get()

	if err != nil {
		return err
	}

	res.PhaseCount = uint8(phases)

	// this can return an error, if phase switching isn't activated via the App
	return nrg.updateControl(res, true)
}

var _ api.PhaseGetter = (*NRGKickGen2)(nil)

func (nrg *NRGKickGen2) GetPhases() (int, error) {
	res, err := nrg.controlG.Get()

	if err != nil {
		return 0, err
	}

	return int(res.PhaseCount), nil
}
