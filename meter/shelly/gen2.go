package shelly

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// Gen2API endpoint reference: https://shelly-api-docs.shelly.cloud/gen2/

type Gen2RpcPost struct {
	Id     int    `json:"id"`
	On     bool   `json:"on"`
	Src    string `json:"src"`
	Method string `json:"method"`
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

var _ Generation = (*gen2)(nil)

type gen2 struct {
	*request.Helper
	uri          string
	channel      int
	model        string
	profile      string
	switchstatus util.Cacheable[Gen2SwitchStatus]
	em1status    func() (Gen2EM1Status, error)
	em1data      func() (Gen2EM1Data, error)
	emstatus     func() (Gen2EMStatus, error)
	emdata       func() (Gen2EMData, error)
}

func apiCall[T any](c *gen2, api string) func() (T, error) {
	return func() (T, error) {
		var res T
		if err := c.execCmd(fmt.Sprintf("%s?id=%d", api, c.channel), false, &res); err != nil {
			return res, err
		}
		return res, nil
	}
}

// gen2InitApi initializes the connection to the shelly gen2+ api and sets up the cached gen2SwitchStatus, gen2EM1Status and gen2EMStatus
func newGen2(helper *request.Helper, uri, model, profile string, channel int, user, password string, cache time.Duration) *gen2 {
	c := &gen2{
		Helper:  helper,
		uri:     uri,
		channel: channel,
		model:   model,
		profile: profile,
	}

	// Shelly GEN 2+ API
	// https://shelly-api-docs.shelly.cloud/gen2/
	c.uri = fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "http"))
	if user != "" {
		c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
	}

	c.switchstatus = util.ResettableCached(apiCall[Gen2SwitchStatus](c, "Switch.GetStatus"), cache)
	c.em1status = util.Cached(apiCall[Gen2EM1Status](c, "EM1.GetStatus"), cache)
	c.em1data = util.Cached(apiCall[Gen2EM1Data](c, "EM1Data.GetStatus"), cache)
	c.emstatus = util.Cached(apiCall[Gen2EMStatus](c, "EM.GetStatus"), cache)
	c.emdata = util.Cached(apiCall[Gen2EMData](c, "EMData.GetStatus"), cache)

	return c
}

// execCmd executes a shelly api gen2+ command and provides the response
func (c *gen2) execCmd(method string, enable bool, res any) error {
	// Shelly gen 2 rfc7616 authentication
	// https://shelly-api-docs.shelly.cloud/gen2/Overview/CommonDeviceTraits#authentication
	// https://datatracker.ietf.org/doc/html/rfc7616

	data := &Gen2RpcPost{
		Id:     c.channel,
		On:     enable,
		Src:    "evcc",
		Method: method,
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
	// Endpoint Switch.GetStatus
	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Apower, err

		// Endpoint EM1.GetStatus
	case c.hasEMEndpoint() && c.profile == "monophase":
		res, err := c.em1status()
		return res.ActPower, err

		// Endpoint EM.GetStatus
	case c.hasEMEndpoint() && c.profile == "triphase":
		res, err := c.emstatus()
		return res.TotalActPower, err

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
	return c.execCmd("Switch.Set?id="+strconv.Itoa(c.channel), enable, &res)
}

// TotalEnergy implements the api.Meter interface
func (c *gen2) TotalEnergy() (float64, error) {
	switch {
	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Aenergy.Total / 1000, err

	case c.hasEMEndpoint() && c.profile == "monophase":
		res, err := c.em1data()
		return res.TotalActEnergy / 1000, err

	case c.hasEMEndpoint() && c.profile == "triphase":
		res, err := c.emdata()
		return res.TotalAct / 1000, err

	default:
		return 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Currents implements the api.PhaseCurrents interface
func (c *gen2) Currents() (float64, float64, float64, error) {
	switch {
	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Current, 0, 0, err

	case c.hasEMEndpoint() && c.profile == "monophase":
		res, err := c.em1status()
		return res.Current, 0, 0, err

	case c.hasEMEndpoint() && c.profile == "triphase":
		res, err := c.emstatus()
		return res.ACurrent, res.BCurrent, res.CCurrent, err

	default:
		return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Voltages implements the api.PhaseVoltages interface
func (c *gen2) Voltages() (float64, float64, float64, error) {
	switch {
	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Voltage, 0, 0, err

	case c.hasEMEndpoint() && c.profile == "monophase":
		res, err := c.em1status()
		return res.Voltage, 0, 0, err

	case c.hasEMEndpoint() && c.profile == "triphase":
		res, err := c.emstatus()
		return res.AVoltage, res.BVoltage, res.CVoltage, err

	default:
		return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Powers implements the api.PhasePowers interface
func (c *gen2) Powers() (float64, float64, float64, error) {
	switch {
	case c.hasSwitchEndpoint():
		res, err := c.switchstatus.Get()
		return res.Apower, 0, 0, err

	case c.hasEMEndpoint() && c.profile == "monophase":
		res, err := c.em1status()
		return res.ActPower, 0, 0, err

	case c.hasEMEndpoint() && c.profile == "triphase":
		res, err := c.emstatus()
		return res.AActPower, res.BActPower, res.CActPower, err

	default:
		return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
	}
}

// Gen2+ models using Switch.GetStatus endpoint https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Switch#switchgetstatus-example
func (c *gen2) hasSwitchEndpoint() bool {
	// Generation 2 Devices (Plus Series):
	// - SNSW-001X16EU: Shelly Plus 1 with 1x relay
	// - SNSW-001P16EU: Shelly Plus 1PM with 1x relay + power meter
	// - SNSW-002P16EU: Shelly Plus 2PM with 2x relay + power meter
	// - SNPL-00112EU: Shelly Plus Plug S (EU)
	// - SNPL-00110IT: Shelly Plus Plug S (Italy)
	// - SNPL-00112UK: Shelly Plus Plug S (UK)
	// - SNPL-00116US: Shelly Plus Plug S (US)
	// Generation 2 Devices (Pro Series - Hutschiene):
	// - SPSW-001XE16EU: Shelly Pro 1 with 1x relay
	// - SPSW-001PE16EU: Shelly Pro 1 PM with 1x relay + power meter
	// - SPSW-002XE16EU: Shelly Pro 2 with 2x relay
	// - SPSW-002PE16EU: Shelly Pro 2 PM with 2x relay + power meter
	// - SPSW-003XE16EU: Shelly Pro 3 with 3x relay
	// - SPSW-004PE16EU: Shelly Pro 4 PM with 4x relay + power meter
	// Generation 3 Devices:
	// - S3SW-001P16EU: Shelly 1PM Gen3 with 1x relay + power meter
	// Generation 4 Devices:
	// - S4SW-001X8EU: Shelly 1 Mini Gen4 with 1x relay
	// - S4SW-001P8EU: Shelly 1PM Mini Gen4 with 1x relay + power meter
	// - S4SW-001P16EU: Shelly 1PM Gen4 with 1x relay + power meter
	switchModels := []string{"SNSW", "SNPL", "SPSW", "S3SW", "S4SW"}
	return slices.Contains(switchModels, c.model)
}

// Gen2+ models using EM1.GetStatus endpoint for power and EM1Data.GetStatus for energy
// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/EM1#em1getstatus-example
// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/EM1Data#em1datagetstatus-example
func (c *gen2) hasEMEndpoint() bool {
	// Generation 2 Devices (Pro Series - Hutschiene):
	// - SPEM-002CEBEU50: Shelly Pro EM 50 with 1x relay + 2x energy meter
	// - SPEM-003CEBEU120: Shelly Pro 3EM with 3x energy meter
	// - SPEM-003CEBEU63: Shelly Pro 3EM-3CT63 with 3x energy meter
	// Generation 3 Devices:
	// - S3EM-002CXCEU: Shelly EM Gen 3 with 1x relay + 2x energy meter
	// - S3EM-001XCEU: Shelly 3EM Gen 3 with 3x energy meter
	// - S3EM-003CXCEU63: Shelly 3EM Gen 3 with 3x energy meter
	// Generation 4 Devices:
	// - S4EM-001PXCEU16: Shelly EM Mini Gen4 with power meter
	em1Models := []string{"SPEM", "S3EM", "S4EM"}
	return slices.Contains(em1Models, c.model)
}
