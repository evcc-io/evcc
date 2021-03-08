package meter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
)

// credits to https://github.com/vloschiavo/powerwall2

const (
	teslaMeterURI   = "/meters/aggregates"
	teslaBatteryURI = "/system_status/soe"
	teslaLoginURI   = "/login/Basic"
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
	uri      *url.URL
	usage    string
	password string
}

func init() {
	registry.Add("tesla", NewTeslaFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -p meter -f decorateTesla -b api.Meter -o tesla_decorators -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,SoC,func() (float64, error)"

// NewTeslaFromConfig creates a Tesla Powerwall Meter from generic config
func NewTeslaFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI, Usage, Password string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage setting")
	}

	if cc.Password == "" {
		return nil, errors.New("missing password setting")
	}

	uri, err := url.Parse(cc.URI)
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

	return NewTesla(uri, cc.Usage, cc.Password)
}

// NewTesla creates a Tesla Meter
func NewTesla(uri *url.URL, usage string, password string) (api.Meter, error) {
	log := util.NewLogger("tesla")

	uri.Scheme = "https"
	// normalize path to '/api' so user input with a path won't cause any trouble
	uri.Path = "/api"

	m := &Tesla{
		Helper:   request.NewHelper(log),
		uri:      uri,
		usage:    strings.ToLower(usage),
		password: password,
	}

	// ignore the self signed certificate
	m.Client.Transport = request.NewTripper(log, request.InsecureTransport())
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
	// username for the powerwall seems to always be customer; email is not required for authentication
	payload := map[string]interface{}{"password": m.password, "username": "customer"}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// returns cookie which is saved in the cookie jar
	resp, err := m.Client.Post(m.uri.String()+teslaLoginURI, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to post: %s", err)
	}

	defer resp.Body.Close()
	// failed to login
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %s", err)
		}
		return fmt.Errorf("couldn't login: %s", body)
	}

	return nil
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Tesla) CurrentPower() (float64, error) {
	var res teslaMeterResponse
	if err := m.GetJSON(m.uri.String()+teslaMeterURI, &res); err != nil {
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
	if err := m.GetJSON(m.uri.String()+teslaMeterURI, &res); err != nil {
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
	err := m.GetJSON(m.uri.String()+teslaBatteryURI, &res)

	return res.Percentage, err
}
