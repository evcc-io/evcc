package charger

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type HomeAssistantSwitch struct {
	baseURL string
	switchE string
	powerE  string
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
		Switch       string
		Power        string
		StandbyPower float64
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeAssistantSwitch(cc.embed, cc.BaseURL, cc.Token, cc.Switch, cc.Power, cc.StandbyPower)
}

func NewHomeAssistantSwitch(embed embed, baseURL, token, switchE, powerE string, standbypower float64) (api.Charger, error) {
	log := util.NewLogger("ha-switch")

	c := &HomeAssistantSwitch{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		switchE: switchE,
		powerE:  powerE,
		Helper:  request.NewHelper(log),
	}

	if switchE == "" {
		return nil, errors.New("missing switch entity")
	}

	// standbypower < 0 ensures that currentPower is never used by the switch socket if not present
	if powerE == "" && standbypower >= 0 {
		return nil, errors.New("missing either power entity or negative standbypower")
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.currentPower, standbypower)
	c.Helper.Client.Transport = &transport.Decorator{
		Decorator: transport.DecorateHeaders(map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		}),
		Base: c.Helper.Client.Transport,
	}

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *HomeAssistantSwitch) Enabled() (bool, error) {
	var res struct {
		State string `json:"state"`
	}

	uri := fmt.Sprintf("%s/api/states/%s", c.baseURL, c.switchE)
	err := c.Helper.GetJSON(uri, &res)

	return res.State == "on", err
}

// Enable implements the api.Charger interface
func (c *HomeAssistantSwitch) Enable(enable bool) error {
	service := "turn_off"
	if enable {
		service = "turn_on"
	}

	data := map[string]any{"entity_id": c.switchE}
	// the domain must not be necessary a 'switch' - it can be also an `input_boolean`
	domain := strings.Split(c.switchE, ".")[0]

	uri := fmt.Sprintf("%s/api/services/%s/%s", c.baseURL, domain, service)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)

	return c.Helper.DoJSON(req, nil)
}

// currentPower implements the api.Meter interface (optional)
func (c *HomeAssistantSwitch) currentPower() (float64, error) {
	var res struct {
		State float64 `json:"state,string"`
	}

	uri := fmt.Sprintf("%s/api/states/%s", c.baseURL, c.powerE)
	err := c.Helper.GetJSON(uri, &res)

	return res.State, err
}
