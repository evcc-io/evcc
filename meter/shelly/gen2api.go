package shelly

// Package shellyapiv2plus implements the Shelly Gen2+ API
import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/evcc-io/evcc/util/request"
)

// Gen2CurrentPower implements the Gen2 api.Meter interface
func (c *Connection) Gen2CurrentPower() (float64, error) {
	// Endpoint Switch.GetStatus
	var switchResponse Gen2SwitchStatus
	if switchEndpointModel(c.model) {
		if err := c.gen2ExecCmd("Switch.GetStatus?id="+strconv.Itoa(c.channel), false, &switchResponse); err != nil {
			return 0, err
		}
		return switchResponse.Apower, nil
	}
	// Endpoint EM1.GetStatus
	var em1Response Gen2EM1Status
	if em1EndpointModel(c.model) && c.profile == "monophase" {
		if err := c.gen2ExecCmd("EM1.GetStatus?id="+strconv.Itoa(c.channel), false, &em1Response); err != nil {
			return 0, err
		}
		return em1Response.ActPower, nil
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
	var switchResponse Gen2SwitchStatus
	if switchEndpointModel(c.model) {
		if err := c.gen2ExecCmd("Switch.GetStatus?id="+strconv.Itoa(c.channel), false, &switchResponse); err != nil {
			return false, err
		}
		return switchResponse.Output, nil
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
	var switchResponse Gen2SwitchStatus
	if switchEndpointModel(c.model) {
		if err := c.gen2ExecCmd("Switch.GetStatus?id="+strconv.Itoa(c.channel), false, &switchResponse); err != nil {
			return 0, 0, err
		}
		energyConsumed = switchResponse.Aenergy.Total
		// Some devices are not providing the Ret_Aenergy information
		// in this case it depends on the installation an we are setting both energy values to the Aenergy total
		if switchResponse.Ret_Aenergy.Total == nil {
			energyFeedIn = switchResponse.Aenergy.Total
		} else {
			energyFeedIn = *switchResponse.Ret_Aenergy.Total
		}
		return energyConsumed, energyFeedIn, nil
	}
	// Endpoint EM1Data.GetStatus (monophase profile)
	var em1DataResponse Gen2EM1Data
	if em1EndpointModel(c.model) && c.profile == "monophase" {
		if err := c.gen2ExecCmd("EM1Data.GetStatus?id="+strconv.Itoa(c.channel), false, &em1DataResponse); err != nil {
			return 0, 0, err
		}
		energyConsumed = em1DataResponse.TotalActEnergy
		energyFeedIn = em1DataResponse.TotalActRetEnergy
		return energyConsumed, energyFeedIn, nil
	}
	// Endpoint EMData.GetStatus (triphase profile)
	var emDataResponse Gen2EMData
	if em1EndpointModel(c.model) && c.profile == "triphase" {
		if err := c.gen2ExecCmd("EMData.GetStatus?id="+strconv.Itoa(c.channel), false, &emDataResponse); err != nil {
			return 0, 0, err
		}
		energyConsumed = emDataResponse.TotalAct
		energyFeedIn = emDataResponse.TotalActRet
		return energyConsumed, energyFeedIn, nil
	}
	return 0, 0, fmt.Errorf("unknown shelly model: %s", c.model)
}

// gen2ExecCmd executes a shelly api gen2+ command and provides the response
func (d *Connection) gen2ExecCmd(method string, enable bool, res interface{}) error {
	// Shelly gen 2 rfc7616 authentication
	// https://shelly-api-docs.shelly.cloud/gen2/Overview/CommonDeviceTraits#authentication
	// https://datatracker.ietf.org/doc/html/rfc7616

	data := &Gen2RpcPost{
		Id:     d.channel,
		On:     enable,
		Src:    "evcc",
		Method: method,
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/%s", d.uri, method), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return d.DoJSON(req, &res)
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
