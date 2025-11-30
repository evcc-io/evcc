package battery

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// API is the HomeWizard Battery API client
type API struct {
	*request.Helper
	host  string
	token string
}

// NewAPI creates a new HomeWizard Battery API client
func NewAPI(host, token string) *API {
	log := util.NewLogger("homewizard-battery").Redact(token)

	a := &API{
		Helper: request.NewHelper(log),
		host:   host,
		token:  token,
	}

	// Use insecure HTTPS transport for self-signed certificates
	a.Client.Transport = transport.Insecure()

	// Set timeout for unreachable devices
	a.Client.Timeout = 10 * time.Second

	return a
}

// apiRequest creates an HTTP request with proper headers
func (a *API) apiRequest(method, endpoint string, body any) (*http.Request, error) {
	uri := fmt.Sprintf("https://%s%s", a.host, endpoint)

	var req *http.Request
	var err error

	if body != nil {
		req, err = request.New(method, uri, request.MarshalJSON(body), request.JSONEncoding)
	} else {
		req, err = request.New(method, uri, nil, request.JSONEncoding)
	}

	if err != nil {
		return nil, err
	}

	// Set required headers for HomeWizard Battery API v2
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("X-Api-Version", "2")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

// GetDeviceInfo retrieves device information to identify the device type
func (a *API) GetDeviceInfo() (DeviceInfo, error) {
	var res DeviceInfo

	req, err := a.apiRequest(http.MethodGet, "/api", nil)
	if err != nil {
		return res, err
	}

	if err := a.DoJSON(req, &res); err != nil {
		return res, err
	}

	return res, nil
}

// GetMeasurement retrieves battery measurement data
func (a *API) GetMeasurement() (Measurement, error) {
	var res Measurement

	req, err := a.apiRequest(http.MethodGet, "/api/measurement", nil)
	if err != nil {
		return res, err
	}

	if err := a.DoJSON(req, &res); err != nil {
		return res, err
	}

	return res, nil
}

// GetBatteries retrieves battery system status from P1 meter
func (a *API) GetBatteries() (Status, error) {
	var res Status

	req, err := a.apiRequest(http.MethodGet, "/api/batteries", nil)
	if err != nil {
		return res, err
	}

	if err := a.DoJSON(req, &res); err != nil {
		return res, err
	}

	return res, nil
}

// SetBatteryMode sets the battery control mode
func (a *API) SetBatteryMode(mode api.BatteryMode) error {
	var hwMode string
	switch mode {
	case api.BatteryNormal:
		hwMode = "zero"
	case api.BatteryCharge:
		hwMode = "to_full"
	case api.BatteryHold:
		hwMode = "standby"
	default:
		return fmt.Errorf("unsupported battery mode: %v", mode)
	}

	reqBody := struct {
		Mode string `json:"mode"`
	}{
		Mode: hwMode,
	}

	req, err := a.apiRequest(http.MethodPut, "/api/batteries", reqBody)
	if err != nil {
		return err
	}

	var res Status
	if err := a.DoJSON(req, &res); err != nil {
		return err
	}

	return nil
}
