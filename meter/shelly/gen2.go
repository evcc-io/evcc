package shelly

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// Gen2API endpoint reference: https://shelly-api-docs.shelly.cloud/gen2/

type Gen2RpcRequest struct {
	Id     int    `json:"id"`
	Src    string `json:"src"`
	Method string `json:"method"`
}

type Gen2SetRpcPost struct {
	Gen2RpcRequest
	On bool `json:"on"`
}

type Gen2Methods struct {
	Methods []string
}

type Gen2SwitchStatus struct {
	Output  bool
	Apower  float64
	Voltage float64
	Current float64
	Aenergy struct {
		Total float64
	}
	Ret_Aenergy struct {
		Total float64
	}
}

type Gen2EMStatus struct {
	TotalActPower float64 `json:"total_act_power"`
	ACurrent      float64 `json:"a_current"`
	BCurrent      float64 `json:"b_current"`
	CCurrent      float64 `json:"c_current"`
	AVoltage      float64 `json:"a_voltage"`
	BVoltage      float64 `json:"b_voltage"`
	CVoltage      float64 `json:"c_voltage"`
	AActPower     float64 `json:"a_act_power"`
	BActPower     float64 `json:"b_act_power"`
	CActPower     float64 `json:"c_act_power"`
}

type Gen2EMData struct {
	TotalAct    float64 `json:"total_act"`
	TotalActRet float64 `json:"total_act_ret"`
}

type Gen2EM1Status struct {
	Current  float64 `json:"current"`
	Voltage  float64 `json:"voltage"`
	ActPower float64 `json:"act_power"`
}

type Gen2EM1Data struct {
	TotalActEnergy    float64 `json:"total_act_energy"`
	TotalActRetEnergy float64 `json:"total_act_ret_energy"`
}

type Gen2ProAddOnGetPeripherals struct {
	DigitalOut map[string]any `json:"digital_out"`
}

var _ Generation = (*gen2)(nil)

const apisrc string = "evcc"

type gen2 struct {
	*request.Helper
	uri           string
	switchchannel int
	model         string
	methods       []string
	switchstatus  util.Cacheable[Gen2SwitchStatus]
	em1status     func() (Gen2EM1Status, error)
	em1data       func() (Gen2EM1Data, error)
	emstatus      func() (Gen2EMStatus, error)
	emdata        func() (Gen2EMData, error)
}

func apiCall[T any](c *gen2, id int, method string) func() (T, error) {
	return func() (T, error) {
		var res T
		if err := c.execCmd(id, method, &res); err != nil {
			return res, err
		}
		return res, nil
	}
}

// gen2InitApi initializes the connection to the shelly gen2+ api and sets up the cached gen2SwitchStatus, gen2EM1Status and gen2EMStatus
func newGen2(helper *request.Helper, uri, model string, channel int, user, password string, cache time.Duration) (*gen2, error) {
	// Shelly GEN 2+ API
	// https://shelly-api-docs.shelly.cloud/gen2/
	c := &gen2{
		Helper:        helper,
		uri:           fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "http")),
		switchchannel: channel,
		model:         model,
	}

	// Shelly gen 2 rfc7616 authentication
	// https://shelly-api-docs.shelly.cloud/gen2/General/Authentication
	if user != "" {
		c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
	}

	var res Gen2Methods
	if err := c.execCmd(channel, "Shelly.ListMethods", &res); err != nil {
		return nil, err
	}

	c.methods = res.Methods

	// Optional change of switchchannel for Pro shellies with peripherals
	if c.hasMethod("ProOutputAddon.GetPeripherals") {
		var err error
		c.switchchannel, err = c.getAddOnSwitchId(channel)
		if err != nil {
			return nil, err
		}
	}

	if c.hasMethod("PM1.GetStatus") {
		c.switchstatus = util.ResettableCached(apiCall[Gen2SwitchStatus](c, channel, "PM1.GetStatus"), cache)
	} else {
		c.switchstatus = util.ResettableCached(apiCall[Gen2SwitchStatus](c, c.switchchannel, "Switch.GetStatus"), cache)
	}
	c.em1status = util.Cached(apiCall[Gen2EM1Status](c, channel, "EM1.GetStatus"), cache)
	c.em1data = util.Cached(apiCall[Gen2EM1Data](c, channel, "EM1Data.GetStatus"), cache)
	c.emstatus = util.Cached(apiCall[Gen2EMStatus](c, channel, "EM.GetStatus"), cache)
	c.emdata = util.Cached(apiCall[Gen2EMData](c, channel, "EMData.GetStatus"), cache)

	return c, nil
}

// execCmd executes a shelly api gen2+ command and provides the response
func (c *gen2) execCmd(id int, method string, res any) error {
	data := &Gen2RpcRequest{
		Id:     id,
		Src:    apisrc,
		Method: method,
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/%s", c.uri, method), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return c.DoJSON(req, &res)
}

// execCmd executes a shelly api gen2+ command and provides the response
func (c *gen2) execEnableCmd(id int, method string, enable bool, res any) error {
	data := &Gen2SetRpcPost{
		Gen2RpcRequest: Gen2RpcRequest{
			Id:     id,
			Src:    apisrc,
			Method: method,
		},
		On: enable,
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/%s", c.uri, method), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return c.DoJSON(req, &res)
}

// CurrentPower implements the api.Meter interface
func (c *gen2) CurrentPower() (float64, error) {
	switch {
	case c.hasEM1Endpoint():
		res, err := c.em1status()
		return res.ActPower, err

	case c.hasEMEndpoint():
		res, err := c.emstatus()
		return res.TotalActPower, err

	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Apower, err

	default:
		return 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Gen2Enabled implements the Gen2 api.Charger interface
func (c *gen2) Enabled() (bool, error) {
	if c.hasSwitchEndpoint() {
		res, err := c.switchstatus.Get()
		return res.Output, err
	}

	return false, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2Enable implements the api.Charger interface
func (c *gen2) Enable(enable bool) error {
	var res Gen2SwitchStatus
	c.switchstatus.Reset()
	return c.execEnableCmd(c.switchchannel, "Switch.Set", enable, &res)
}

// TotalEnergy implements the api.Meter interface
func (c *gen2) TotalEnergy() (float64, error) {
	switch {
	case c.hasEM1Endpoint():
		res, err := c.em1data()
		return res.TotalActEnergy / 1000, err

	case c.hasEMEndpoint():
		res, err := c.emdata()
		return res.TotalAct / 1000, err

	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Aenergy.Total / 1000, err

	default:
		return 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Currents implements the api.PhaseCurrents interface
func (c *gen2) Currents() (float64, float64, float64, error) {
	switch {
	case c.hasEM1Endpoint():
		res, err := c.em1status()
		return res.Current, 0, 0, err

	case c.hasEMEndpoint():
		res, err := c.emstatus()
		return res.ACurrent, res.BCurrent, res.CCurrent, err

	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Current, 0, 0, err

	default:
		return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Voltages implements the api.PhaseVoltages interface
func (c *gen2) Voltages() (float64, float64, float64, error) {
	switch {
	case c.hasEM1Endpoint():
		res, err := c.em1status()
		return res.Voltage, 0, 0, err

	case c.hasEMEndpoint():
		res, err := c.emstatus()
		return res.AVoltage, res.BVoltage, res.CVoltage, err

	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Voltage, 0, 0, err

	default:
		return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Powers implements the api.PhasePowers interface
func (c *gen2) Powers() (float64, float64, float64, error) {
	switch {
	case c.hasEM1Endpoint():
		res, err := c.em1status()
		return res.ActPower, 0, 0, err

	case c.hasEMEndpoint():
		res, err := c.emstatus()
		return res.AActPower, res.BActPower, res.CActPower, err

	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Apower, 0, 0, err

	default:
		return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Gen2+ models using Switch.GetStatus endpoint https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Switch#switchgetstatus-example
func (c *gen2) hasSwitchEndpoint() bool {
	return c.hasMethod("Switch.GetStatus") || c.hasMethod("PM1.GetStatus")
}

func (c *gen2) hasEM1Endpoint() bool {
	return c.hasMethod("EM1.GetStatus")
}

func (c *gen2) hasEMEndpoint() bool {
	return c.hasMethod("EM.GetStatus")
}

// Gen2+ models using EM1.GetStatus endpoint for power and EM1Data.GetStatus for energy
// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/EM1#em1getstatus-example
// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/EM1Data#em1datagetstatus-example
func (c *gen2) hasMethod(method string) bool {
	return slices.Contains(c.methods, method)
}

func (c *gen2) getAddOnSwitchId(channel int) (int, error) {
	var res Gen2ProAddOnGetPeripherals
	if err := c.execCmd(channel, "ProOutputAddon.GetPeripherals", &res); err != nil {
		return channel, err
	}

	return parseAddOnSwitchID(channel, res), nil
}

func parseAddOnSwitchID(channel int, res Gen2ProAddOnGetPeripherals) int {
	if _, ok := res.DigitalOut["switch:100"]; ok {
		return 100
	}

	// if no switch ID is found, return the channel as default
	return channel
}
