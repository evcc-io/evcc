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
	log   *util.Logger
	host  string
	token string
}

// NewAPI creates a new HomeWizard Battery API client
func NewAPI(host, token string) *API {
	log := util.NewLogger("homewizard-battery").Redact(token)

	api := &API{
		Helper: request.NewHelper(log),
		log:    log,
		host:   host,
		token:  token,
	}

	// Use insecure HTTPS transport for self-signed certificates
	api.Client.Transport = transport.Insecure()

	// Set timeout for unreachable devices
	api.Client.Timeout = 10 * time.Second

	return api
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

	a.log.TRACE.Printf("Request: %s %s", method, uri)
	a.log.TRACE.Printf("Headers: Authorization=Bearer <redacted>, X-Api-Version=2, Content-Type=%s, Accept=%s",
		req.Header.Get("Content-Type"), req.Header.Get("Accept"))

	return req, nil
}

// GetDeviceInfo retrieves device information to identify the device type
func (a *API) GetDeviceInfo() (DeviceInfo, error) {
	var res DeviceInfo

	req, err := a.apiRequest(http.MethodGet, "/api", nil)
	if err != nil {
		a.log.ERROR.Printf("failed to create device info request: %v", err)
		return res, err
	}

	if err := a.DoJSON(req, &res); err != nil {
		a.log.ERROR.Printf("failed to get device info: %v", err)
		return res, err
	}

	a.log.DEBUG.Printf("device info: %s (%s) - firmware %s, API %s",
		res.ProductName, res.ProductType, res.FirmwareVersion, res.APIVersion)

	return res, nil
}

// GetMeasurement retrieves battery measurement data
func (a *API) GetMeasurement() (Measurement, error) {
	var res Measurement

	req, err := a.apiRequest(http.MethodGet, "/api/measurement", nil)
	if err != nil {
		a.log.ERROR.Printf("failed to create measurement request: %v", err)
		return res, err
	}

	if err := a.DoJSON(req, &res); err != nil {
		a.log.ERROR.Printf("failed to get measurement: %v", err)
		return res, err
	}

	return res, nil
}

// GetBatteries retrieves battery system status from P1 meter
func (a *API) GetBatteries() (Status, error) {
	var res Status

	req, err := a.apiRequest(http.MethodGet, "/api/batteries", nil)
	if err != nil {
		a.log.ERROR.Printf("failed to create batteries request: %v", err)
		return res, err
	}

	if err := a.DoJSON(req, &res); err != nil {
		a.log.ERROR.Printf("failed to get batteries: %v", err)
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

	a.log.DEBUG.Printf("setting battery mode: %s -> %s", mode, hwMode)

	reqBody := struct {
		Mode string `json:"mode"`
	}{
		Mode: hwMode,
	}

	req, err := a.apiRequest(http.MethodPut, "/api/batteries", reqBody)
	if err != nil {
		a.log.ERROR.Printf("failed to create battery mode request: %v", err)
		return err
	}

	a.log.DEBUG.Printf("PUT https://%s/api/batteries with mode=%s", a.host, hwMode)

	var res Status
	if err := a.DoJSON(req, &res); err != nil {
		a.log.ERROR.Printf("failed to set battery mode: %v", err)
		return err
	}

	a.log.DEBUG.Printf("battery mode set successfully to: %s", res.Mode)
	return nil
}
