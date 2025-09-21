package charger

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/evcc-io/evcc/api"
    "github.com/evcc-io/evcc/util"
    "github.com/evcc-io/evcc/util/request"
)

// AvalonQ charger implementation
type AvalonQ struct {
    *request.Helper
    uri      string
    workmode string
    power    map[string]float64
}

// AvalonQ API response structures
type AvalonQStatus struct {
    WorkMode string  `json:"work_mode"`
    Power    float64 `json:"power_consumption"`
    Enabled  bool    `json:"enabled"`
}

type AvalonQResponse struct {
    Success bool          `json:"success"`
    Data    AvalonQStatus `json:"data"`
}

func init() {
    registry.Add("avalonq", NewAvalonQFromConfig)
}

// NewAvalonQFromConfig creates a AvalonQ charger from generic config
func NewAvalonQFromConfig(other map[string]interface{}) (api.Charger, error) {
    cc := struct {
        URI string `mapstructure:"uri"`
    }{}

    if err := util.DecodeOther(other, &cc); err != nil {
        return nil, err
    }

    return NewAvalonQ(cc.URI)
}

// NewAvalonQ creates AvalonQ charger
func NewAvalonQ(uri string) (api.Charger, error) {
    c := &AvalonQ{
        Helper: request.NewHelper(util.NewLogger("avalonq")),
        uri:    util.DefaultScheme(uri, "http"),
        power: map[string]float64{
            "eco":         800.0,
            "standard":    1300.0,
            "performance": 1700.0,
        },
    }

    return c, nil
}

// Status implements the api.Charger interface
func (c *AvalonQ) Status() (api.ChargeStatus, error) {
    status, err := c.getStatus()
    if err != nil {
        return api.StatusNone, err
    }

    if !status.Enabled {
        return api.StatusA, nil // Not charging
    }

    return api.StatusC, nil // Charging
}

// Enabled implements the api.Charger interface
func (c *AvalonQ) Enabled() (bool, error) {
    status, err := c.getStatus()
    if err != nil {
        return false, err
    }

    return status.Enabled, nil
}

// Enable implements the api.Charger interface
func (c *AvalonQ) Enable(enable bool) error {
    var workmode string
    if enable {
        if c.workmode == "" {
            workmode = "standard" // Default workmode
        } else {
            workmode = c.workmode
        }
    } else {
        workmode = "off"
    }

    return c.setWorkMode(workmode)
}

// MaxCurrent implements the api.Charger interface
func (c *AvalonQ) MaxCurrent() (int64, error) {
    status, err := c.getStatus()
    if err != nil {
        return 0, err
    }

    // Convert workmode to "current" level (1-3)
    switch status.WorkMode {
    case "eco":
        return 1, nil
    case "standard":
        return 2, nil
    case "performance":
        return 3, nil
    default:
        return 0, nil
    }
}

// SetMaxCurrent implements the api.Charger interface
func (c *AvalonQ) SetMaxCurrent(current int64) error {
    var workmode string

    switch current {
    case 0:
        workmode = "off"
    case 1:
        workmode = "eco"
    case 2:
        workmode = "standard"
    case 3:
        workmode = "performance"
    default:
        workmode = "standard"
    }

    c.workmode = workmode
    return c.setWorkMode(workmode)
}

// ChargePower implements the api.Meter interface
func (c *AvalonQ) ChargePower() (float64, error) {
    status, err := c.getStatus()
    if err != nil {
        return 0, err
    }

    if !status.Enabled {
        return 0, nil
    }

    return status.Power, nil
}

// getStatus retrieves miner status from API
func (c *AvalonQ) getStatus() (AvalonQStatus, error) {
    var status AvalonQResponse
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.uri+"/api/v1/status", nil)
    if err != nil {
        return AvalonQStatus{}, err
    }

    resp, err := c.DoJSON(req, &status)
    if err != nil {
        return AvalonQStatus{}, err
    }

    if resp.StatusCode != http.StatusOK || !status.Success {
        return AvalonQStatus{}, fmt.Errorf("api error: status %d", resp.StatusCode)
    }

    return status.Data, nil
}

// setWorkMode sets miner work mode via API
func (c *AvalonQ) setWorkMode(mode string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    data := map[string]string{"work_mode": mode}
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.uri+"/api/v1/workmode", 
        bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    var response AvalonQResponse
    resp, err := c.DoJSON(req, &response)
    if err != nil {
        return err
    }

    if resp.StatusCode != http.StatusOK || !response.Success {
        return fmt.Errorf("failed to set workmode: status %d", resp.StatusCode)
    }

    return nil
}
