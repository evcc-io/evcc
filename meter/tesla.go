package meter

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/powerwall"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// credits to https://github.com/vloschiavo/powerwall2

// Tesla is the tesla powerwall meter
type Tesla struct {
	*request.Helper
	uri, usage, password string
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateTesla -b *Tesla -r api.Meter -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,SoC,func() (float64, error)"

// NewTeslaFromConfig creates a Tesla Powerwall Meter from generic config
func NewTeslaFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI, Usage, Password string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	if cc.Password == "" {
		return nil, errors.New("missing password")
	}

	_, err := url.Parse(cc.URI)
	if err != nil {
		return nil, fmt.Errorf("%s is invalid: %s", cc.URI, err)
	}

	// support default meter names
	switch strings.ToLower(cc.Usage) {
	case "grid":
		cc.Usage = "site"
	case "pv":
		cc.Usage = "solar"
	}

	return NewTesla(cc.URI, cc.Usage, cc.Password)
}

// NewTesla creates a Tesla Meter
func NewTesla(uri, usage, password string) (api.Meter, error) {
	log := util.NewLogger("tesla").Redact(password)

	m := &Tesla{
		Helper:   request.NewHelper(log),
		uri:      util.DefaultScheme(strings.TrimSuffix(uri, "/"), "https"),
		usage:    strings.ToLower(usage),
		password: password,
	}

	// ignore the self signed certificate
	m.Client.Transport = request.NewTripper(log, transport.Insecure())
	// create cookie jar to save login tokens
	m.Client.Jar, _ = cookiejar.New(nil)

	if err := m.Login(); err != nil {
		return nil, err
	}

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

// Login calls login and saves the returned cookie
func (m *Tesla) Login() error {
	data := map[string]interface{}{
		"username": "customer",
		"password": m.password,
	}

	req, err := request.New(http.MethodPost, m.uri+powerwall.LoginURI, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		// use DoBody as it will close the response body
		if _, err = m.DoBody(req); err != nil {
			err = fmt.Errorf("login failed: %w", err)
		}
	}

	return err
}

var _ api.Meter = (*Tesla)(nil)

// CurrentPower implements the api.Meter interface
func (m *Tesla) CurrentPower() (float64, error) {
	var res powerwall.MeterResponse
	if err := m.GetJSON(m.uri+powerwall.MeterURI, &res); err != nil {
		return 0, err
	}

	if o, ok := res[m.usage]; ok {
		return o.InstantPower, nil
	}

	return 0, fmt.Errorf("invalid usage: %s", m.usage)
}

// totalEnergy implements the api.MeterEnergy interface
func (m *Tesla) totalEnergy() (float64, error) {
	var res powerwall.MeterResponse
	if err := m.GetJSON(m.uri+powerwall.MeterURI, &res); err != nil {
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
	var res powerwall.BatteryResponse
	err := m.GetJSON(m.uri+powerwall.BatteryURI, &res)

	return res.Percentage, err
}
