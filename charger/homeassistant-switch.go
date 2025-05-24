package charger

import (
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type HomeAssistantSwitch struct {
	baseURL      string
	token        string
	switchEntity string
	powerEntity  string
	helper       *request.Helper
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
	helper := request.NewHelper(util.NewLogger("homeassistant-switch"))
	helper.Client.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		}),
		Base: helper.Client.Transport,
	}

	c := &HomeAssistantSwitch{
		baseURL:      baseURL,
		token:        token,
		switchEntity: switchEntity,
		powerEntity:  powerEntity,
		helper:       helper,
	}
	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.CurrentPower, standbypower)
	return c, nil
}

func (c *HomeAssistantSwitch) apiRequest(method, path string, body interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	var req *http.Request
	var err error
	headers := map[string]string{"Content-Type": "application/json"}
	if body != nil {
		req, err = request.New(method, url, request.MarshalJSON(body), headers)
	} else {
		req, err = request.New(method, url, nil, headers)
	}
	if err != nil {
		return nil, err
	}
	return c.helper.DoBody(req)
}

// Enabled implements the api.Charger interface
func (c *HomeAssistantSwitch) Enabled() (bool, error) {
	path := fmt.Sprintf("%s/api/states/%s", c.baseURL, c.switchEntity)
	var resp struct {
		State string `json:"state"`
	}
	if err := c.helper.GetJSON(path, &resp); err != nil {
		return false, err
	}
	return resp.State == "on", nil
}

// Enable implements the api.Charger interface
func (c *HomeAssistantSwitch) Enable(enable bool) error {
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
	path := fmt.Sprintf("%s/api/states/%s", c.baseURL, c.powerEntity)
	var resp struct {
		State string `json:"state"`
	}
	if err := c.helper.GetJSON(path, &resp); err != nil {
		return 0, err
	}
	var val float64
	_, err := fmt.Sscanf(resp.State, "%f", &val)
	return val, err
}
