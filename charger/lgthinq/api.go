package lgthinq

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// ThinQ Connect API, see https://smartsolution.lge.com/thinq-connect (PAT at https://connect-pat.lgthinq.com)

// ApiKey is the public ThinQ Connect api key
const ApiKey = "v6GFvkweNo7DK7yD3ylIZ9w52aKBU0eJ7wLXkSR3"

const DeviceTypeWaterHeater = "DEVICE_WATER_HEATER"

var (
	kic = []string{"AU", "BD", "CN", "HK", "ID", "IN", "JP", "KH", "KR", "LA", "LK", "MM", "MY", "NP", "NZ", "PH", "SG", "TH", "TW", "VN"}
	aic = []string{"AG", "AR", "AW", "BB", "BO", "BR", "BS", "BZ", "CA", "CL", "CO", "CR", "CU", "DM", "DO", "EC", "GD", "GT", "GY", "HN", "HT", "JM", "KN", "LC", "MX", "NI", "PA", "PE", "PR", "PY", "SR", "SV", "TT", "US", "UY", "VC", "VE"}
)

// region returns the api region for the given ISO 3166-1 alpha-2 country code
func region(country string) string {
	switch {
	case slices.Contains(kic, country):
		return "kic"
	case slices.Contains(aic, country):
		return "aic"
	default:
		return "eic"
	}
}

func randomId() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

type Device struct {
	DeviceId   string `json:"deviceId"`
	DeviceInfo struct {
		DeviceType string `json:"deviceType"`
		ModelName  string `json:"modelName"`
		Alias      string `json:"alias"`
		Reportable bool   `json:"reportable"`
	} `json:"deviceInfo"`
}

type response struct {
	Response any `json:"response"`
	Error    struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type API struct {
	*request.Helper
	base string
}

// NewAPI creates a ThinQ Connect api client using a personal access token
func NewAPI(log *util.Logger, token, country string) *API {
	country = strings.ToUpper(country)

	api := &API{
		Helper: request.NewHelper(log),
		base:   fmt.Sprintf("https://api-%s.lgthinq.com", region(country)),
	}

	clientId := "evcc-" + randomId()

	api.Client.Transport = &transport.Decorator{
		Base: api.Client.Transport,
		Decorator: func(req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("x-api-key", ApiKey)
			req.Header.Set("x-client-id", clientId)
			req.Header.Set("x-country", country)
			req.Header.Set("x-message-id", randomId())
			req.Header.Set("x-service-phase", "OP")
			return nil
		},
	}

	return api
}

func (api *API) do(method, path string, body, res any) error {
	var data io.Reader
	headers := request.AcceptJSON
	if body != nil {
		data = request.MarshalJSON(body)
		headers = request.JSONEncoding
	}

	req, err := request.New(method, api.base+path, data, headers)
	if err != nil {
		return err
	}

	var resp response
	if res != nil {
		resp.Response = res
	}

	if err := api.DoJSON(req, &resp); err != nil {
		if resp.Error.Code != "" {
			return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return err
	}

	return nil
}

// Devices returns the list of devices
func (api *API) Devices() ([]Device, error) {
	var res []Device
	err := api.do(http.MethodGet, "/devices", nil, &res)
	return res, err
}

// State returns the device state
func (api *API) State(deviceId string, res any) error {
	return api.do(http.MethodGet, "/devices/"+deviceId+"/state", nil, res)
}

// Control sends a control command to the device
func (api *API) Control(deviceId string, body any) error {
	return api.do(http.MethodPost, "/devices/"+deviceId+"/control", body, nil)
}

// WaterHeaterState is the state of a DEVICE_WATER_HEATER
type WaterHeaterState struct {
	WaterHeaterJobMode struct {
		CurrentJobMode string `json:"currentJobMode"`
	} `json:"waterHeaterJobMode"`
	Operation struct {
		WaterHeaterOperationMode string `json:"waterHeaterOperationMode"`
	} `json:"operation"`
	TemperatureInUnits []Temperature `json:"temperatureInUnits"`
}

type Temperature struct {
	CurrentTemperature float64 `json:"currentTemperature"`
	TargetTemperature  float64 `json:"targetTemperature"`
	Unit               string  `json:"unit"`
}

// Celsius returns the celsius temperature
func (s WaterHeaterState) Celsius() (Temperature, bool) {
	for _, t := range s.TemperatureInUnits {
		if t.Unit == "C" {
			return t, true
		}
	}
	return Temperature{}, false
}

// WaterHeaterTargetTemperature sets the water heater target temperature in celsius
func (api *API) WaterHeaterTargetTemperature(deviceId string, temp float64) error {
	return api.Control(deviceId, map[string]any{
		"temperatureInUnits": map[string]any{
			"targetTemperature": temp,
			"unit":              "C",
		},
	})
}
