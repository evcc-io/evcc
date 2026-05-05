package leapmotor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	BaseURL       = "https://appgateway.leapmotor-international.de"
	appVersion    = "1.12.3"
	source        = "leapmotor"
	channel       = "1"
	deviceType    = "1"
	p12EncAlg     = "1"
	policyID      = "20260204"
	defaultLang   = "en"
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
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
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
