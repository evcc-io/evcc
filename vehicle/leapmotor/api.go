package leapmotor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util/request"
)

const (
	BaseURL     = "https://appgateway.leapmotor-international.de"
	appVersion  = "1.12.3"
	source      = "leapmotor"
	channel     = "1"
	deviceType  = "1"
	p12EncAlg   = "1"
	policyID    = "20260204"
	defaultLang = "en"
)

type apiEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// LoginResponse holds fields returned by the login endpoint.
type LoginResponse struct {
	ID           json.Number `json:"id"`
	UID          string      `json:"uid"`
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
	SignIkm      string      `json:"signIkm"`
	SignSalt     string      `json:"signSalt"`
	SignInfo     string      `json:"signInfo"`
	Base64Cert   string      `json:"base64Cert"`
}

// Vehicle is a vehicle entry from the account vehicle list.
type Vehicle struct {
	VIN     string `json:"vin"`
	CarType string `json:"carType"`
}

// StatusData holds the vehicle status fields relevant to EVCC.
type StatusData struct {
	Soc              *int     `json:"soc"`
	ChargeState      *int     `json:"chargeState"`
	ChargeRemainTime *int     `json:"chargeRemainTime"`
	BatteryCurrent   *float64 `json:"batteryCurrent"`
	BatteryVoltage   *float64 `json:"batteryVoltage"`
	ExpectedMileage  *int     `json:"expectedMileage"`
	Speed            *int     `json:"speed"`
	TotalMileage     *int     `json:"totalMileage"`
	ChargeSocSetting *int     `json:"chargesocSetting"`
	AcSwitch         *bool    `json:"acSwitch"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
}

// apiPost sends a POST to fullURL with the given headers and body, returns the response body.
func apiPost(client *http.Client, fullURL string, headers map[string]string, body string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, fullURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return (&request.Helper{Client: client}).DoBody(req)
}

// parseEnvelope decodes the API envelope, returning Data or an error for non-zero codes.
func parseEnvelope[T any](body []byte) (T, error) {
	var res apiEnvelope[T]
	var zero T
	if err := json.Unmarshal(body, &res); err != nil {
		return zero, err
	}
	if res.Code != 0 {
		return zero, fmt.Errorf("api %d: %s", res.Code, res.Message)
	}
	return res.Data, nil
}

// postAndParse sends a POST and decodes the API envelope in one step.
func postAndParse[T any](client *http.Client, fullURL string, headers map[string]string, body string) (T, error) {
	b, err := apiPost(client, fullURL, headers, body)
	if err != nil {
		var zero T
		return zero, err
	}
	return parseEnvelope[T](b)
}

// signalToField maps the numeric signal IDs returned by C10/B10 (and other
// signal-based models) to the named StatusData fields. T03 returns these
// fields flat, so it bypasses this mapping. IDs from the verified APK table.
const (
	sigSoc              = "1204"
	sigChargeState      = "1149"
	sigChargeRemainTime = "1200"
	sigBatteryCurrent   = "1178"
	sigBatteryVoltage   = "1177"
	sigExpectedMileage  = "3260"
	sigSpeed            = "1319"
	sigTotalMileage     = "1318"
	sigAcSwitch         = "1938"
	sigLatitude         = "3725"
	sigLongitude        = "3724"
	sigLatitudeAlt      = "2190"
	sigLongitudeAlt     = "2191"
)

// parseStatusData decodes a status response. T03 returns flat fields; C10/B10
// nest telemetry under "signal" (numeric IDs) and the charge limit under
// config.3.percent. Flat fields take priority so T03 passes through unchanged.
func parseStatusData(body []byte) (StatusData, error) {
	raw, err := parseEnvelope[json.RawMessage](body)
	if err != nil {
		return StatusData{}, err
	}

	var sd StatusData
	if err := json.Unmarshal(raw, &sd); err != nil {
		return StatusData{}, err
	}

	var nested struct {
		Signal map[string]json.Number `json:"signal"`
		Config map[string]struct {
			Percent *int `json:"percent"`
		} `json:"config"`
	}
	if err := json.Unmarshal(raw, &nested); err != nil || nested.Signal == nil {
		return sd, nil // flat (T03) response
	}

	sig := nested.Signal
	setIfNil(&sd.Soc, sigInt(sig, sigSoc))
	setIfNil(&sd.ChargeState, sigInt(sig, sigChargeState))
	setIfNil(&sd.ChargeRemainTime, sigInt(sig, sigChargeRemainTime))
	setIfNilF(&sd.BatteryCurrent, sigFloat(sig, sigBatteryCurrent))
	setIfNilF(&sd.BatteryVoltage, sigFloat(sig, sigBatteryVoltage))
	setIfNil(&sd.ExpectedMileage, sigInt(sig, sigExpectedMileage))
	setIfNil(&sd.Speed, sigInt(sig, sigSpeed))
	setIfNil(&sd.TotalMileage, sigInt(sig, sigTotalMileage))
	if sd.AcSwitch == nil {
		sd.AcSwitch = sigBool(sig, sigAcSwitch)
	}
	setIfNilF(&sd.Latitude, firstFloat(sig, sigLatitude, sigLatitudeAlt))
	setIfNilF(&sd.Longitude, firstFloat(sig, sigLongitude, sigLongitudeAlt))

	if sd.ChargeSocSetting == nil {
		if c, ok := nested.Config["3"]; ok && c.Percent != nil {
			sd.ChargeSocSetting = c.Percent
		}
	}

	return sd, nil
}

func setIfNil(dst **int, v *int) {
	if *dst == nil {
		*dst = v
	}
}

func setIfNilF(dst **float64, v *float64) {
	if *dst == nil {
		*dst = v
	}
}

func sigInt(sig map[string]json.Number, id string) *int {
	if n, ok := sig[id]; ok {
		if f, err := n.Float64(); err == nil {
			i := int(f)
			return &i
		}
	}
	return nil
}

func sigFloat(sig map[string]json.Number, id string) *float64 {
	if n, ok := sig[id]; ok {
		if f, err := n.Float64(); err == nil {
			return &f
		}
	}
	return nil
}

func sigBool(sig map[string]json.Number, id string) *bool {
	if n, ok := sig[id]; ok {
		if f, err := n.Float64(); err == nil {
			b := f != 0
			return &b
		}
	}
	return nil
}

func firstFloat(sig map[string]json.Number, ids ...string) *float64 {
	for _, id := range ids {
		if v := sigFloat(sig, id); v != nil {
			return v
		}
	}
	return nil
}
