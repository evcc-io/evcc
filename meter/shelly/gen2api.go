package shelly

// Package shellyapiv2plus implements the Shelly Gen2+ API
import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// Gen2CurrentPower implements the Gen2 api.Meter interface
func (c *Connection) Gen2CurrentPower() (float64, error) {
	// Endpoint Switch.GetStatus
	if switchEndpointModel(c.model) {
		res, err := c.gen2SwitchStatus.Get()
		return res.Apower, err
	}
	// Endpoint EM1.GetStatus
	if em1EndpointModel(c.model) && c.profile == "monophase" {
		res, err := c.gen2EM1Status.Get()
		return res.ActPower, err
	}
	// Endpoint EM.GetStatus
	var emResponse Gen2EMStatus
	if em1EndpointModel(c.model) && c.profile == "triphase" {
		if err := c.gen2ExecCmd("EM.GetStatus?id="+strconv.Itoa(c.channel), false, &emResponse); err != nil {
			return 0, err
		}
		return emResponse.TotalActPower, nil
	}
	return 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2Enabled implements the Gen2 api.Charger interface
func (c *Connection) Gen2Enabled() (bool, error) {
	if switchEndpointModel(c.model) {
		res, err := c.gen2SwitchStatus.Get()
		return res.Output, err
	}
	return false, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2Enable implements the api.Charger interface
func (c *Connection) Gen2Enable(enable bool) error {
	var res Gen2SwitchStatus
	return c.gen2ExecCmd("Switch.Set", enable, &res)
}

// Gen2TotalEnergy implements the api.Meter interface
func (c *Connection) Gen2TotalEnergy() (float64, float64, error) {
	var energyConsumed float64
	var energyFeedIn float64
	// Endpoint Switch.GetStatus
	if switchEndpointModel(c.model) {
		res, err := c.gen2SwitchStatus.Get()
		if err != nil {
			return 0, 0, err
		}
		energyConsumed = res.Aenergy.Total
		// Some devices are not providing the Ret_Aenergy information
		// in this case it depends on the installation an we are setting both energy values to the Aenergy total
		if res.Ret_Aenergy.Total == nil {
			energyFeedIn = res.Aenergy.Total
		} else {
			energyFeedIn = *res.Ret_Aenergy.Total
		}
		return energyConsumed / 1000, energyFeedIn / 1000, nil
	}
	// Endpoint EM1Data.GetStatus (monophase profile)
	var em1DataResponse Gen2EM1Data
	if em1EndpointModel(c.model) && c.profile == "monophase" {
		if err := c.gen2ExecCmd("EM1Data.GetStatus?id="+strconv.Itoa(c.channel), false, &em1DataResponse); err != nil {
			return 0, 0, err
		}
		energyConsumed = em1DataResponse.TotalActEnergy
		energyFeedIn = em1DataResponse.TotalActRetEnergy
		return energyConsumed / 1000, energyFeedIn / 1000, nil
	}
	// Endpoint EMData.GetStatus (triphase profile)
	var emDataResponse Gen2EMData
	if em1EndpointModel(c.model) && c.profile == "triphase" {
		if err := c.gen2ExecCmd("EMData.GetStatus?id="+strconv.Itoa(c.channel), false, &emDataResponse); err != nil {
			return 0, 0, err
		}
		energyConsumed = emDataResponse.TotalAct
		energyFeedIn = emDataResponse.TotalActRet
		return energyConsumed / 1000, energyFeedIn / 1000, nil
	}
	return 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2Currents implements the api.PhaseCurrents interface
func (c *Connection) Gen2Currents() (float64, float64, float64, error) {
	// Endpoint Switch.GetStatus
	if switchEndpointModel(c.model) {
		res, err := c.gen2SwitchStatus.Get()
		if err != nil {
			return 0, 0, 0, err
		}
		return res.Current, 0, 0, nil
	}
	// Endpoint EM1.GetStatus
	if em1EndpointModel(c.model) && c.profile == "monophase" {
		res, err := c.gen2EM1Status.Get()
		return res.Current, 0, 0, err
	}
	// Endpoint EM.GetStatus
	var emResponse Gen2EMStatus
	if em1EndpointModel(c.model) && c.profile == "triphase" {
		if err := c.gen2ExecCmd("EM.GetStatus?id="+strconv.Itoa(c.channel), false, &emResponse); err != nil {
			return 0, 0, 0, err
		}
		return emResponse.ACurrent, emResponse.BCurrent, emResponse.CCurrent, nil
	}
	return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2Voltages implements the api.PhaseVoltages interface
func (c *Connection) Gen2Voltages() (float64, float64, float64, error) {
	// Endpoint Switch.GetStatus
	if switchEndpointModel(c.model) {
		res, err := c.gen2SwitchStatus.Get()
		if err != nil {
			return 0, 0, 0, err
		}
		return res.Voltage, 0, 0, nil
	}
	// Endpoint EM1.GetStatus
	if em1EndpointModel(c.model) && c.profile == "monophase" {
		res, err := c.gen2EM1Status.Get()
		return res.Voltage, 0, 0, err
	}
	// Endpoint EM.GetStatus
	var emResponse Gen2EMStatus
	if em1EndpointModel(c.model) && c.profile == "triphase" {
		if err := c.gen2ExecCmd("EM.GetStatus?id="+strconv.Itoa(c.channel), false, &emResponse); err != nil {
			return 0, 0, 0, err
		}
		return emResponse.AVoltage, emResponse.BVoltage, emResponse.CVoltage, nil
	}
	return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// Gen2Powers implements the api.PhasePowers interface
func (c *Connection) Gen2Powers() (float64, float64, float64, error) {
	// Endpoint Switch.GetStatus
	if switchEndpointModel(c.model) {
		res, err := c.gen2SwitchStatus.Get()
		if err != nil {
			return 0, 0, 0, err
		}
		return res.Apower, 0, 0, nil
	}
	// Endpoint EM1.GetStatus
	if em1EndpointModel(c.model) && c.profile == "monophase" {
		res, err := c.gen2EM1Status.Get()
		return res.ActPower, 0, 0, err
	}
	// Endpoint EM.GetStatus
	var emResponse Gen2EMStatus
	if em1EndpointModel(c.model) && c.profile == "triphase" {
		if err := c.gen2ExecCmd("EM.GetStatus?id="+strconv.Itoa(c.channel), false, &emResponse); err != nil {
			return 0, 0, 0, err
		}
		return emResponse.AActPower, emResponse.BActPower, emResponse.CActPower, nil
	}
	return 0, 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// gen2ExecCmd executes a shelly api gen2+ command and provides the response
func (c *Connection) gen2ExecCmd(method string, enable bool, res interface{}) error {
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

func (c *Connection) gen2InitApi(uri, user, password string) {
	// Shelly GEN 2+ API
	// https://shelly-api-docs.shelly.cloud/gen2/
	c.uri = fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "http"))
	if user != "" {
		c.Client.Transport = digest.NewTransport(user, password, c.Client.Transport)
	}
	// Cached gen2SwitchStatus
	c.gen2SwitchStatus = util.ResettableCached(func() (Gen2SwitchStatus, error) {
		var gen2SwitchStatusResponse Gen2SwitchStatus
		err := c.gen2ExecCmd("Switch.GetStatus?id="+strconv.Itoa(c.channel), false, &gen2SwitchStatusResponse)
		if err != nil {
			return Gen2SwitchStatus{}, err
		}
		return gen2SwitchStatusResponse, nil
	}, c.Cache)
	// Cached gen2EM1Status
	c.gen2EM1Status = util.ResettableCached(func() (Gen2EM1Status, error) {
		var gen2EM1StatusResponse Gen2EM1Status
		err := c.gen2ExecCmd("EM1.GetStatus?id="+strconv.Itoa(c.channel), false, &gen2EM1StatusResponse)
		if err != nil {
			return Gen2EM1Status{}, err
		}
		return gen2EM1StatusResponse, nil
	}, c.Cache)
}

// Gen2+ models using Switch.GetStatus endpoint https://shelly-api-docs.shelly.cloud/gen2/ComponentsAndServices/Switch#switchgetstatus-example
func switchEndpointModel(model string) bool {
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
	// - S3SW-001P16EU: Shelly 1 PM Gen3 with 1x relay + power meter
	switchModels := []string{"SNSW", "SNPL", "SPSW", "S3SW"}
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
func em1EndpointModel(model string) bool {
	// Generation 2 Devices (Pro Series - Hutschiene):
	// - SPEM-002CEBEU50: Shelly Pro EM 50 with 1x relay + 2x energy meter
	// - SPEM-003CEBEU120: Shelly Pro 3EM with 3x energy meter
	// - SPEM-003CEBEU63: Shelly Pro 3EM-3CT63 with 3x energy meter
	// Generation 3 Devices:
	// - S3EM-002CXCEU: Shelly EM Gen 3 with 1x relay + 2x energy meter
	// - S3EM-001XCEU: Shelly 3EM Gen 3 with 3x energy meter
	// - S3EM-003CXCEU63: Shelly 3EM Gen 3 with 3x energy meter
	em1Models := []string{"SPEM", "S3EM"}
	for _, em1Model := range em1Models {
		if em1Model == model {
			return true
		}
	}
	return false
}
