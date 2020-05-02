package meter

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

// credits to https://github.com/vloschiavo/powerwall2

type teslaResponse map[string]struct {
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

// Tesla is the tesla powerwall meter
type Tesla struct {
	*util.HTTPHelper
	uri, usage string
}

// NewTeslaFromConfig creates a Tesla Powerwall Meter from generic config
func NewTeslaFromConfig(log *util.Logger, other map[string]interface{}) api.Meter {
	cc := struct {
		URI, Usage string
	}{}
	util.DecodeOther(log, other, &cc)

	if cc.Usage == "" {
		log.FATAL.Fatalf("config: missing usage")
	}

	url, err := url.ParseRequestURI(cc.URI)
	if err != nil {
		log.FATAL.Fatalf("config: invalid uri %s", cc.URI)
	}

	if url.Path == "" {
		url.Path = "api/meters/aggregates"
		cc.URI = url.String()
	}

	return NewTesla(cc.URI, cc.Usage)
}

// NewTesla creates a Tesla Meter
func NewTesla(uri, usage string) api.Meter {
	m := &Tesla{
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("tsla")),
		uri:        uri,
		usage:      strings.ToLower(usage),
	}

	// ignore the self signed certificate
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	m.HTTPHelper.Client.Transport = customTransport

	// decorate api.MeterEnergy
	if m.usage == "load" || m.usage == "solar" {
		return &TeslaEnergy{Tesla: m}
	}

	return m
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Tesla) CurrentPower() (float64, error) {
	var tr teslaResponse
	_, err := m.GetJSON(m.uri, &tr)

	if err == nil {
		if o, ok := tr[m.usage]; ok {
			return o.InstantPower, nil
		}
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// TeslaEnergy decorates Tesla with api.MeterEnergy interface
type TeslaEnergy struct {
	*Tesla
}

// TotalEnergy implements the api.MeterEnergy interface
func (m *TeslaEnergy) TotalEnergy() (float64, error) {
	var tr teslaResponse
	_, err := m.GetJSON(m.uri, &tr)

	if err == nil {
		if o, ok := tr[m.usage]; ok {
			if m.usage == "load" {
				return o.EnergyImported, nil
			}
			if m.usage == "solar" {
				return o.EnergyExported, nil
			}
		}
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}
