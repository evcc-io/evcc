package charger

// LICENSE: MIT
//
// Lektrico charger protocol (discovered from lektricowifi source + real device testing):
//
//   GET  http://<host>/rpc/<Method>   -> direct JSON response
//   POST http://<host>/rpc            -> JSON-RPC body, response wrapped in "result" field
//
// Main endpoint: charger_info.get returns all data in a single request.
//
// Example response from charger_info.get:
//   {
//     "charger_state": "B",             // raw IEC state: A/B/C/D/E/F/B_AUTH/B_PAUSE/OTA/LOCKED
//     "extended_charger_state": "B_AUTH",
//     "session_energy": 38.48,          // Wh
//     "instant_power": 0.0,             // W
//     "currents": [0.0, 0.0, 0.0],      // A, array [L1, L2, L3]
//     "voltages": [237.65, 0.0, 0.0],   // V, array [L1, L2, L3]
//     "total_charged_energy": 9683.844, // kWh
//     "dynamic_current": 32,            // allowed current (0=pause, 6-32=active)
//     "has_active_errors": false,
//     "charger_is_paused": false,
//     "current_limit_reason": 2,        // int (0=no_limit, 2=user_limit, ...)
//     "temperature": 18.8,
//     "fw_version": "1.51",
//     "headless": true,                 // true = no authentication required
//     "install_current": 32,
//   }

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("lektrico", NewLektricoFromConfig)
}

// Raw IEC states returned in charger_state / extended_charger_state
const (
	lektricoIECA          = "A"           // no vehicle connected
	lektricoIECB          = "B"           // vehicle connected, not charging
	lektricoIECBAUTH      = "B_AUTH"      // waiting for RFID authentication
	lektricoIECBPAUSE     = "B_PAUSE"     // charging paused (dynamic_current=0)
	lektricoIECBSCHEDULER = "B_SCHEDULER" // paused by schedule
	lektricoIECC          = "C"           // charging active
	lektricoIECD          = "D"           // charging active with ventilation
	lektricoIECE          = "E"           // error
	lektricoIECF          = "F"           // fatal error
	lektricoIECOTA        = "OTA"         // firmware update in progress
	lektricoIECLOCKED     = "LOCKED"      // charger locked
)

// lektricoInfo maps the JSON response from charger_info.get
type lektricoInfo struct {
	// State
	ChargerState         string `json:"charger_state"`          // current IEC state
	ExtendedChargerState string `json:"extended_charger_state"` // detailed state (B_AUTH, B_PAUSE, etc.)
	ChargerIsPaused      bool   `json:"charger_is_paused"`
	HasActiveErrors      bool   `json:"has_active_errors"`

	// Energy & power
	InstantPower       float64   `json:"instant_power"`        // W
	SessionEnergy      float64   `json:"session_energy"`       // Wh
	TotalChargedEnergy float64   `json:"total_charged_energy"` // kWh
	Currents           []float64 `json:"currents"`             // [L1, L2, L3] in A
	Voltages           []float64 `json:"voltages"`             // [L1, L2, L3] in V

	// Configuration
	DynamicCurrent     int     `json:"dynamic_current"`      // 0=pause, 6-32=allowed current
	InstallCurrent     int     `json:"install_current"`      // max installed current
	CurrentLimitReason int     `json:"current_limit_reason"` // 0=no_limit, 2=user_limit, ...
	Temperature        float64 `json:"temperature"`
	FwVersion          string  `json:"fw_version"`
	Headless           bool    `json:"headless"` // true = no authentication required
}

// lektricoRPCRequest is the POST JSON-RPC request format
type lektricoRPCRequest struct {
	Src    string         `json:"src"`
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// lektricoRPCResponse wraps the POST response (data is in the "result" field)
type lektricoRPCResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Lektrico implements api.Charger for Lektrico 1P7K / 3P22K charging stations
type Lektrico struct {
	log     *util.Logger
	baseURL string
	client  *http.Client
	current int64 // last valid current, stored for Enable()
}

// LektricoConfig is the YAML configuration
type LektricoConfig struct {
	Host string `mapstructure:"host"`
}

// NewLektricoFromConfig creates a Lektrico instance from evcc configuration
func NewLektricoFromConfig(other map[string]any) (api.Charger, error) {
	var cc LektricoConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.Host == "" {
		return nil, fmt.Errorf("lektrico: missing 'host' parameter (e.g. 192.168.1.100)")
	}
	return NewLektrico(cc.Host)
}

// NewLektrico creates a Lektrico charger and verifies connectivity
func NewLektrico(host string) (*Lektrico, error) {
	l := &Lektrico{
		log:     util.NewLogger("lektrico"),
		baseURL: fmt.Sprintf("http://%s/rpc", host),
		client:  &http.Client{Timeout: 10 * time.Second},
		current: 6,
	}

	// Connectivity check via Device_id.Get (minimal response, fast)
	var id struct {
		DeviceID string `json:"device_id"`
	}
	if err := l.get("Device_id.Get", &id); err != nil {
		return nil, fmt.Errorf("lektrico: connection failed to %s: %w", host, err)
	}
	l.log.DEBUG.Printf("connected to Lektrico charger: %s (fw %s)", id.DeviceID, l.fwVersion())

	return l, nil
}

// fwVersion silently reads the firmware version for logging (best-effort)
func (l *Lektrico) fwVersion() string {
	var info lektricoInfo
	if err := l.get("charger_info.get", &info); err != nil {
		return "unknown"
	}
	return info.FwVersion
}

// get performs GET http://<host>/rpc/<uri> and decodes the JSON response directly
func (l *Lektrico) get(uri string, result any) error {
	url := l.baseURL + "/" + uri
	resp, err := l.client.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", uri, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", uri, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("GET %s read: %w", uri, err)
	}

	l.log.TRACE.Printf("GET /%s -> %s", uri, string(body))
	return json.Unmarshal(body, result)
}

// post performs POST http://<host>/rpc with a JSON-RPC body
// The charger returns {"id":..., "result":{...}} - we check for errors only
func (l *Lektrico) post(method string, params map[string]any) error {
	payload := lektricoRPCRequest{
		Src:    "evcc",
		ID:     rand.Intn(90000000) + 10000000,
		Method: method,
		Params: params,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	l.log.TRACE.Printf("POST /rpc %s %v", method, params)

	resp, err := l.client.Post(l.baseURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST %s: %w", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %s: HTTP %d", method, resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	l.log.TRACE.Printf("POST /rpc %s -> %s", method, string(respBody))

	var rpcResp lektricoRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("POST %s decode: %w", method, err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("POST %s charger error: code=%d msg=%s",
			method, rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return nil
}

// info reads charger_info.get - a single request returns all data
func (l *Lektrico) info() (lektricoInfo, error) {
	var info lektricoInfo
	err := l.get("charger_info.get", &info)
	return info, err
}

// Status implements api.Charger - returns IEC 61851 charge status (A/B/C/E)
func (l *Lektrico) Status() (api.ChargeStatus, error) {
	info, err := l.info()
	if err != nil {
		return api.StatusNone, err
	}

	if info.HasActiveErrors {
		return api.StatusE, nil
	}

	// Use extended_charger_state which is more precise than charger_state
	switch info.ExtendedChargerState {
	case lektricoIECA:
		return api.StatusA, nil // no vehicle
	case lektricoIECB, lektricoIECBAUTH, lektricoIECBPAUSE,
		lektricoIECBSCHEDULER, lektricoIECLOCKED:
		return api.StatusB, nil // connected but not charging
	case lektricoIECC, lektricoIECD:
		return api.StatusC, nil // charging active
	case lektricoIECE, lektricoIECF:
		return api.StatusE, nil
	case lektricoIECOTA:
		return api.StatusB, nil // firmware update in progress
	default:
		l.log.WARN.Printf("unknown IEC state: %q", info.ExtendedChargerState)
		return api.StatusA, nil
	}
}

// Enabled implements api.Charger - returns true if dynamic_current >= 6 (charging allowed)
func (l *Lektrico) Enabled() (bool, error) {
	info, err := l.info()
	if err != nil {
		return false, err
	}
	return info.DynamicCurrent >= 6, nil
}

// Enable implements api.Charger - enables or suspends charging via dynamic_current
func (l *Lektrico) Enable(enable bool) error {
	var value int64
	if enable {
		value = l.current
		if value < 6 {
			value = 6
		}
		l.log.DEBUG.Printf("Enable -> dynamic_current=%dA", value)
	} else {
		value = 0
		l.log.DEBUG.Printf("Disable -> dynamic_current=0 (pause)")
	}
	return l.post("dynamic_current.set", map[string]any{
		"dynamic_current": value,
	})
}

// MaxCurrent implements api.Charger - sets the maximum allowed charge current in amps
func (l *Lektrico) MaxCurrent(current int64) error {
	if current > 32 {
		current = 32
	}
	var value int64
	if current >= 6 {
		value = current
		l.current = current // stored for Enable()
	} else {
		value = 0 // below IEC minimum -> pause
	}
	l.log.DEBUG.Printf("MaxCurrent -> dynamic_current=%dA", value)
	return l.post("dynamic_current.set", map[string]any{
		"dynamic_current": value,
	})
}

// CurrentPower implements api.Meter - returns instant power in Watts
func (l *Lektrico) CurrentPower() (float64, error) {
	info, err := l.info()
	return info.InstantPower, err
}

// TotalEnergy implements api.MeterEnergy - returns total charged energy in kWh
func (l *Lektrico) TotalEnergy() (float64, error) {
	info, err := l.info()
	return info.TotalChargedEnergy, err
}

// ChargedEnergy implements api.ChargeRater - returns session energy in kWh
func (l *Lektrico) ChargedEnergy() (float64, error) {
	info, err := l.info()
	return info.SessionEnergy / 1000.0, err // Wh -> kWh
}

// Currents implements api.PhaseCurrents - returns L1, L2, L3 currents in amps
func (l *Lektrico) Currents() (float64, float64, float64, error) {
	info, err := l.info()
	if err != nil {
		return 0, 0, 0, err
	}
	if len(info.Currents) < 3 {
		return 0, 0, 0, fmt.Errorf("lektrico: incomplete currents array (%d elements)", len(info.Currents))
	}
	return info.Currents[0], info.Currents[1], info.Currents[2], nil
}

// Voltages implements api.PhaseVoltages - returns L1, L2, L3 voltages in volts
func (l *Lektrico) Voltages() (float64, float64, float64, error) {
	info, err := l.info()
	if err != nil {
		return 0, 0, 0, err
	}
	if len(info.Voltages) < 3 {
		return 0, 0, 0, fmt.Errorf("lektrico: incomplete voltages array (%d elements)", len(info.Voltages))
	}
	return info.Voltages[0], info.Voltages[1], info.Voltages[2], nil
}
