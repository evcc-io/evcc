package shelly

import (
	"fmt"
	"net/http"
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

type gen2 struct {
	*request.Helper
	uri          string
	channel      int
	model        string
	profile      string
	switchstatus util.Cacheable[Gen2SwitchStatus]
	em1status    util.Cacheable[Gen2EM1Status]
	em1data      util.Cacheable[Gen2EM1Data]
	emstatus     util.Cacheable[Gen2EMStatus]
	emdata       util.Cacheable[Gen2EMData]
}

// gen2ExecCmd executes a shelly api gen2+ command and provides the response
func (c *gen2) gen2ExecCmd(method string, enable bool, res any) error {
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

// gen2InitApi initializes the connection to the shelly gen2+ api and sets up the cached gen2SwitchStatus, gen2EM1Status and gen2EMStatus
func (c *gen2) InitApi(uri, user, password string, cache time.Duration) {
	// Shelly GEN 2+ API
	// https://shelly-api-docs.shelly.cloud/gen2/
	c.uri = fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "http"))
	if user != "" {
		c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
	}
	// Cached Gen2StatusGen2SwitchStatus
	c.switchstatus = util.ResettableCached(func() (Gen2SwitchStatus, error) {
		var gen2SwitchStatus Gen2SwitchStatus
		if err := c.gen2ExecCmd("Switch.GetStatus?id="+strconv.Itoa(c.channel), false, &gen2SwitchStatus); err != nil {
			return Gen2SwitchStatus{}, err
		}
		return gen2SwitchStatus, nil
	}, cache)
	// Cached gen2EM1Status
	c.em1status = util.ResettableCached(func() (Gen2EM1Status, error) {
		var gen2EM1Status Gen2EM1Status
		if err := c.gen2ExecCmd("EM1.GetStatus?id="+strconv.Itoa(c.channel), false, &gen2EM1Status); err != nil {
			return Gen2EM1Status{}, err
		}
		return gen2EM1Status, nil
	}, cache)
	// Cached gen2EM1Data
	c.em1data = util.ResettableCached(func() (Gen2EM1Data, error) {
		var gen2EM1Data Gen2EM1Data
		if err := c.gen2ExecCmd("EM1Data.GetStatus?id="+strconv.Itoa(c.channel), false, &gen2EM1Data); err != nil {
			return Gen2EM1Data{}, err
		}
		return gen2EM1Data, nil
	}, cache)
	// Cached gen2EMStatus
	c.emstatus = util.ResettableCached(func() (Gen2EMStatus, error) {
		var gen2EMStatus Gen2EMStatus
		if err := c.gen2ExecCmd("EM.GetStatus?id="+strconv.Itoa(c.channel), false, &gen2EMStatus); err != nil {
			return Gen2EMStatus{}, err
		}
		return gen2EMStatus, nil
	}, cache)
	// Cached gen2EMData
	c.emdata = util.ResettableCached(func() (Gen2EMData, error) {
		var gen2EMData Gen2EMData
		if err := c.gen2ExecCmd("EMData.GetStatus?id="+strconv.Itoa(c.channel), false, &gen2EMData); err != nil {
			return Gen2EMData{}, err
		}
		return gen2EMData, nil
	}, cache)
}

// CurrentPower implements the api.Meter interface
func (c *gen2) CurrentPower() (float64, error) {
	// Endpoint Switch.GetStatus
	if hasSwitchEndpoint(c.model) {
		res, err := c.switchstatus.Get()
		return res.Apower, err
	}
	// Endpoint EM1.GetStatus
	if hasEMEndpoint(c.model) && c.profile == "monophase" {
		res, err := c.em1status.Get()
		return res.ActPower, err
	}
	// Endpoint EM.GetStatus
	if hasEMEndpoint(c.model) && c.profile == "triphase" {
		res, err := c.emstatus.Get()
		return res.TotalActPower, err
	}
	return 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2Enabled implements the Gen2 api.Charger interface
func (c *gen2) Enabled() (bool, error) {
	if hasSwitchEndpoint(c.model) {
		res, err := c.switchstatus.Get()
		return res.Output, err
	}
	return false, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2Enable implements the api.Charger interface
func (c *gen2) Enable(enable bool) error {
	var res Gen2SwitchStatus
	c.switchstatus.Reset()
	return c.gen2ExecCmd("Switch.Set?id="+strconv.Itoa(c.channel), enable, &res)
}

// TotalEnergy implements the api.Meter interface
func (c *gen2) TotalEnergy() (float64, error) {
	if hasSwitchEndpoint(c.model) {
		res, err := c.switchstatus.Get()
		if err != nil {
			return 0, err
		}
		return res.Aenergy.Total / 1000, nil
	}
	if hasEMEndpoint(c.model) && c.profile == "monophase" {
		res, err := c.em1data.Get()
		if err != nil {
			return 0, err
		}
		return res.TotalActEnergy / 1000, nil
	}
	if hasEMEndpoint(c.model) && c.profile == "triphase" {
		res, err := c.emdata.Get()
		if err != nil {
			return 0, err
		}
		return res.TotalAct / 1000, nil
	}
	return 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Currents implements the api.PhaseCurrents interface
func (c *gen2) Currents() (float64, float64, float64, error) {
	if hasSwitchEndpoint(c.model) {
		res, err := c.switchstatus.Get()
		if err != nil {
			return 0, 0, 0, err
		}
		return res.Current, 0, 0, nil
	}
	if hasEMEndpoint(c.model) && c.profile == "monophase" {
		res, err := c.em1status.Get()
		return res.Current, 0, 0, err
	}
	if hasEMEndpoint(c.model) && c.profile == "triphase" {
		res, err := c.emstatus.Get()
		return res.ACurrent, res.BCurrent, res.CCurrent, err
	}
	return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Voltages implements the api.PhaseVoltages interface
func (c *gen2) Voltages() (float64, float64, float64, error) {
	if hasSwitchEndpoint(c.model) {
		res, err := c.switchstatus.Get()
		if err != nil {
			return 0, 0, 0, err
		}
		return res.Voltage, 0, 0, nil
	}
	if hasEMEndpoint(c.model) && c.profile == "monophase" {
		res, err := c.em1status.Get()
		return res.Voltage, 0, 0, err
	}
	if hasEMEndpoint(c.model) && c.profile == "triphase" {
		res, err := c.emstatus.Get()
		return res.AVoltage, res.BVoltage, res.CVoltage, err
	}
	return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Powers implements the api.PhasePowers interface
func (c *gen2) Powers() (float64, float64, float64, error) {
	if hasSwitchEndpoint(c.model) {
		res, err := c.switchstatus.Get()
		if err != nil {
			return 0, 0, 0, err
		}
		return res.Apower, 0, 0, nil
	}
	if hasEMEndpoint(c.model) && c.profile == "monophase" {
		res, err := c.em1status.Get()
		return res.ActPower, 0, 0, err
	}
	if hasEMEndpoint(c.model) && c.profile == "triphase" {
		res, err := c.emstatus.Get()
		return res.AActPower, res.BActPower, res.CActPower, err
	}
	return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2+ models using Switch.GetStatus endpoint https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Switch#switchgetstatus-example
func hasSwitchEndpoint(model string) bool {
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
	for _, switchModel := range switchModels {
		if switchModel == model {
			return true
		}
	}
	return false
}

// Gen2+ models using EM1.GetStatus endpoint for power and EM1Data.GetStatus for energy
// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/EM1#em1getstatus-example
// https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/EM1Data#em1datagetstatus-example
func hasEMEndpoint(model string) bool {
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
	for _, em1Model := range em1Models {
		if em1Model == model {
			return true
		}
	}
	return false
}
