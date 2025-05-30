package charger

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type HomeAssistantSwitch struct {
	baseURL      string
	switchEntity string
	powerEntity  string
	*request.Helper
	*switchSocket
}

func init() {
	registry.Add("homeassistant-switch", NewHomeAssistantSwitchFromConfig)
}

func NewHomeAssistantSwitchFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		embed        `mapstructure:",squash"`
		BaseURL      string
		Token        string
		SwitchEntity string
		PowerEntity  string
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeAssistantSwitch(cc.embed, cc.BaseURL, cc.Token, cc.SwitchEntity, cc.PowerEntity, cc.StandbyPower)
}

//go:generate go tool decorate -f decorateHomeAssistantSwitch -b *HomeAssistantSwitch -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)"

func NewHomeAssistantSwitch(embed embed, baseURL, token, switchEntity, powerEntity string, standbypower float64) (api.Charger, error) {
	c := &HomeAssistantSwitch{
		baseURL:      baseURL,
		switchEntity: switchEntity,
		powerEntity:  powerEntity,
		Helper:       request.NewHelper(util.NewLogger("ha-switch")),
	}

	if switchEntity == "" {
		return nil, errors.New("missing switch entity")
	}
	if powerEntity == "" {
		return nil, errors.New("missing power entity")
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.currentPower, standbypower)
	c.Helper.Client.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		}),
		Base: c.Helper.Client.Transport,
	}

	return decorateHomeAssistantSwitch(c, c.currentPower), nil
}

// Enabled implements the api.Charger interface
func (c *HomeAssistantSwitch) Enabled() (bool, error) {
	var res struct {
		State string `json:"state"`
	}

	uri := fmt.Sprintf("%s/api/states/%s", c.baseURL, c.switchEntity)
	err := c.Helper.GetJSON(uri, &res)

	return res.State == "on", err
}

// Enable implements the api.Charger interface
func (c *HomeAssistantSwitch) Enable(enable bool) error {
	service := "turn_off"
	if enable {
		service = "turn_on"
	}

	data := map[string]any{"entity_id": c.switchEntity}

	uri := fmt.Sprintf("%s/api/services/switch/%s", c.baseURL, service)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	return c.Helper.DoJSON(req, nil)
}

// currentPower implements the api.Meter interface (optional)
func (c *HomeAssistantSwitch) currentPower() (float64, error) {
	var res struct {
		State float64 `json:"state,string"`
	}

	uri := fmt.Sprintf("%s/api/states/%s", c.baseURL, c.powerEntity)
	err := c.Helper.GetJSON(uri, &res)

	return res.State, err
}
