package meter

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

func init() {
	registry.Add("tq-em420", NewTqEm420FromConfig)
}

type TqEm420Data struct {
	SmartMeter struct {
		ConfigurationID string `json:"configuration_id"`
		Status          string `json:"status"`
		Timestamp       struct {
			Seconds float64 `json:"seconds"`
			Nanos   float64 `json:"nanos"`
		} `json:"timestamp"`
		Values struct {
			ActiveEnergyP     float64 `json:"active_energy_+"`
			ActiveEnergyL1P   float64 `json:"active_energy_+_L1"`
			ActiveEnergyL2P   float64 `json:"active_energy_+_L2"`
			ActiveEnergyL3P   float64 `json:"active_energy_+_L3"`
			ActiveEnergyM     float64 `json:"active_energy_-"`
			ActiveEnergyL1M   float64 `json:"active_energy_-_L1"`
			ActiveEnergyL2M   float64 `json:"active_energy_-_L2"`
			ActiveEnergyL3M   float64 `json:"active_energy_-_L3"`
			ActivePowerP      float64 `json:"active_power_+"`
			ActivePowerL1P    float64 `json:"active_power_+_L1"`
			ActivePowerL2P    float64 `json:"active_power_+_L2"`
			ActivePowerL3P    float64 `json:"active_power_+_L3"`
			ActivePowerM      float64 `json:"active_power_-"`
			ActivePowerL1M    float64 `json:"active_power_-_L1"`
			ActivePowerL2M    float64 `json:"active_power_-_L2"`
			ActivePowerL3M    float64 `json:"active_power_-_L3"`
			ApparentEnergyP   float64 `json:"apparent_energy_+"`
			ApparentEnergyL1P float64 `json:"apparent_energy_+_L1"`
			ApparentEnergyL2P float64 `json:"apparent_energy_+_L2"`
			ApparentEnergyL3P float64 `json:"apparent_energy_+_L3"`
			ApparentEnergyM   float64 `json:"apparent_energy_-"`
			ApparentEnergyL1M float64 `json:"apparent_energy_-_L1"`
			ApparentEnergyL2M float64 `json:"apparent_energy_-_L2"`
			ApparentEnergyL3M float64 `json:"apparent_energy_-_L3"`
			ApparentPowerP    float64 `json:"apparent_power_+"`
			ApparentPowerL1P  float64 `json:"apparent_power_+_L1"`
			ApparentPowerL2P  float64 `json:"apparent_power_+_L2"`
			ApparentPowerL3P  float64 `json:"apparent_power_+_L3"`
			ApparentPowerM    float64 `json:"apparent_power_-"`
			ApparentPowerL1M  float64 `json:"apparent_power_-_L1"`
			ApparentPowerL2M  float64 `json:"apparent_power_-_L2"`
			ApparentPowerL3M  float64 `json:"apparent_power_-_L3"`
			CurrentL1         float64 `json:"current_L1"`
			CurrentL2         float64 `json:"current_L2"`
			CurrentL3         float64 `json:"current_L3"`
			PowerFactor       float64 `json:"power_factor"`
			PowerFactorL1     float64 `json:"power_factor_L1"`
			PowerFactorL2     float64 `json:"power_factor_L2"`
			PowerFactorL3     float64 `json:"power_factor_L3"`
			ReactiveEnergyP   float64 `json:"reactive_energy_+"`
			ReactiveEnergyL1P float64 `json:"reactive_energy_+_L1"`
			ReactiveEnergyL2P float64 `json:"reactive_energy_+_L2"`
			ReactiveEnergyL3P float64 `json:"reactive_energy_+_L3"`
			ReactiveEnergyM   float64 `json:"reactive_energy_-"`
			ReactiveEnergyL1M float64 `json:"reactive_energy_-_L1"`
			ReactiveEnergyL2M float64 `json:"reactive_energy_-_L2"`
			ReactiveEnergyL3M float64 `json:"reactive_energy_-_L3"`
			ReactivePowerP    float64 `json:"reactive_power_+"`
			ReactivePowerL1P  float64 `json:"reactive_power_+_L1"`
			ReactivePowerL2P  float64 `json:"reactive_power_+_L2"`
			ReactivePowerL3P  float64 `json:"reactive_power_+_L3"`
			ReactivePowerM    float64 `json:"reactive_power_-"`
			ReactivePowerL1M  float64 `json:"reactive_power_-_L1"`
			ReactivePowerL2M  float64 `json:"reactive_power_-_L2"`
			ReactivePowerL3M  float64 `json:"reactive_power_-_L3"`
			SupplyFrequency   float64 `json:"supply_frequency"`
			VoltageL1         float64 `json:"voltage_L1"`
			VoltageL2         float64 `json:"voltage_L2"`
			VoltageL3         float64 `json:"voltage_L3"`
		} `json:"values"`
	} `json:"smart-meter"`
}

type TqEM420 struct {
	dataG func() (TqEm420Data, error)
}

// NewTqEm420FromConfig creates a new configurable meter
func NewTqEm420FromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI    string
		Token  string
		Device string
		Cache  time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("tq-em420").Redact(cc.Token)

	client := request.NewHelper(log)
	client.Jar, _ = cookiejar.New(nil)

	base := util.DefaultScheme(strings.TrimRight(cc.URI, "/"), "http")

	dataG := provider.Cached(func() (TqEm420Data, error) {
		var res TqEm420Data

		headers := map[string]string{
			"Accept":        "application/json",
			"Authorization": "Bearer " + cc.Token,
		}

		req, err := request.New(http.MethodGet, fmt.Sprintf("%s/api/json/"+cc.Device+"/values/smart-meter", base), nil, headers)

		if err == nil {
			err = client.DoJSON(req, &res)
		}

		return res, err
	}, cc.Cache)

	m := &TqEM420{
		dataG: dataG,
	}

	_, err := dataG()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *TqEM420) CurrentPower() (float64, error) {
	res, err := m.dataG()
	return (res.SmartMeter.Values.ActivePowerP - res.SmartMeter.Values.ActivePowerM) / 1e3, err
}

var _ api.MeterEnergy = (*TqEM420)(nil)

func (m *TqEM420) TotalEnergy() (float64, error) {
	res, err := m.dataG()
	return res.SmartMeter.Values.ActiveEnergyP / 1e3, err
}

var _ api.PhaseCurrents = (*TqEM420)(nil)

func (m *TqEM420) Currents() (float64, float64, float64, error) {
	res, err := m.dataG()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.SmartMeter.Values.CurrentL1 / 1e3, res.SmartMeter.Values.CurrentL2 / 1e3, res.SmartMeter.Values.CurrentL3 / 1e3, nil
}
