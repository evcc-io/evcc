package meter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// credits to https://github.com/vloschiavo/powerwall2

const (
	teslaMeterURI   = "/api/meters/aggregates"
	teslaBatteryURI = "/api/system_status/soe"
)

type teslaMeterResponse map[string]struct {
	LastCommunicationTime string  `json:"last_communication_time"`
	InstantPower          float64 `json:"instant_power"`
	InstantReactivePower  float64 `json:"instant_reactive_power"`
	InstantApparentPower  float64 `json:"instant_apparent_power"`
	Frequency             float64 `json:"frequency"`
	EnergyExported        float64 `json:"energy_exported"`
	EnergyImported        float64 `json:"energy_imported"`
	InstantAverageVoltage float64 `json:"instant_average_voltage"`
	InstantTotalCurrent   float64 `json:"instant_total_current"`
	IACurrent             float64 `json:"i_a_current"`
	IBCurrent             float64 `json:"i_b_current"`
	ICCurrent             float64 `json:"i_c_current"`
}

type teslaBatteryResponse struct {
	Percentage float64 `json:"percentage"`
}

// Tesla is the tesla powerwall meter
type Tesla struct {
	*request.Helper
	uri, usage string
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -p meter -f decorateTesla -b api.Meter -o tesla_decorators -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,SoC,func() (float64, error)"

// NewTeslaFromConfig creates a Tesla Powerwall Meter from generic config
func NewTeslaFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI, Usage string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage setting")
	}

	// support default meter names
	switch strings.ToLower(cc.Usage) {
	case "grid":
		cc.Usage = "site"
	case "pv":
		cc.Usage = "solar"
	}

	return NewTesla(cc.URI, cc.Usage)
}

// NewTesla creates a Tesla Meter
func NewTesla(uri, usage string) (api.Meter, error) {
	log := util.NewLogger("tesla")

	m := &Tesla{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(uri, "https"),
		usage:  strings.ToLower(usage),
	}

	// ignore the self signed certificate
	m.Client.Transport = request.NewTripper(log, request.InsecureTransport())

	// decorate api.MeterEnergy
	var totalEnergy func() (float64, error)
	if m.usage == "load" || m.usage == "solar" {
		totalEnergy = m.totalEnergy
	}

	// decorate api.BatterySoC
	var batterySoC func() (float64, error)
	if usage == "battery" {
		batterySoC = m.batterySoC
	}

	return decorateTesla(m, totalEnergy, batterySoC), nil
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Tesla) CurrentPower() (float64, error) {
	var res teslaMeterResponse
	if err := m.GetJSON(m.uri+teslaMeterURI, &res); err != nil {
		return 0, err
	}

	if o, ok := res[m.usage]; ok {
		return o.InstantPower, nil
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// totalEnergy implements the api.MeterEnergy interface
func (m *Tesla) totalEnergy() (float64, error) {
	var res teslaMeterResponse
	if err := m.GetJSON(m.uri+teslaMeterURI, &res); err != nil {
		return 0, err
	}

	if o, ok := res[m.usage]; ok {
		if m.usage == "load" {
			return o.EnergyImported, nil
		}
		if m.usage == "solar" {
			return o.EnergyExported, nil
		}
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// batterySoC implements the api.Battery interface
func (m *Tesla) batterySoC() (float64, error) {
	var res teslaBatteryResponse
	err := m.GetJSON(m.uri+teslaBatteryURI, &res)

	return res.Percentage, err
}
