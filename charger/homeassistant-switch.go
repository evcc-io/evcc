package charger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type HomeAssistantSwitch struct {
	baseURL      string
	token        string
	switchEntity string
	powerEntity  string
	client       *http.Client
	*switchSocket
}

func init() {
	registry.Add("homeassistant-switch", NewHomeAssistantSwitchFromConfig)
}

type haSwitchConfig struct {
	embed        `mapstructure:",squash"`
	BaseURL      string
	Token        string
	SwitchEntity string
	PowerEntity  string
	StandbyPower float64
}

func NewHomeAssistantSwitchFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc haSwitchConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	return NewHomeAssistantSwitch(cc.embed, cc.BaseURL, cc.Token, cc.SwitchEntity, cc.PowerEntity, cc.StandbyPower)
}

func NewHomeAssistantSwitch(embed embed, baseURL, token, switchEntity, powerEntity string, standbypower float64) (*HomeAssistantSwitch, error) {
	c := &HomeAssistantSwitch{
		baseURL:      baseURL,
		token:        token,
		switchEntity: switchEntity,
		powerEntity:  powerEntity,
		client:       &http.Client{Timeout: 5 * time.Second},
	}
	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.CurrentPower, standbypower)
	return c, nil
}

func (c *HomeAssistantSwitch) apiRequest(method, path string, body interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("homeassistantswitch: %s %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// Enabled implements the api.Charger interface
func (c *HomeAssistantSwitch) Enabled() (bool, error) {
	// GET /api/states/<entity_id>
	path := fmt.Sprintf("/api/states/%s", c.switchEntity)
	b, err := c.apiRequest("GET", path, nil)
	if err != nil {
		return false, err
	}
	var resp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return false, err
	}
	return resp.State == "on", nil
}

// Enable implements the api.Charger interface
func (c *HomeAssistantSwitch) Enable(enable bool) error {
	// POST /api/services/switch/turn_on or turn_off
	service := "turn_off"
	if enable {
		service = "turn_on"
	}
	path := fmt.Sprintf("/api/services/switch/%s", service)
	body := map[string]interface{}{"entity_id": c.switchEntity}
	_, err := c.apiRequest("POST", path, body)
	return err
}

// CurrentPower implements the api.Meter interface (optional)
func (c *HomeAssistantSwitch) CurrentPower() (float64, error) {
	if c.powerEntity == "" {
		return 0, nil
	}
	path := fmt.Sprintf("/api/states/%s", c.powerEntity)
	b, err := c.apiRequest("GET", path, nil)
	if err != nil {
		return 0, err
	}
	var resp struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return 0, err
	}
	var val float64
	_, err = fmt.Sscanf(resp.State, "%f", &val)
	return val, err
}
