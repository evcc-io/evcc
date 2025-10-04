package meter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/tess1o/go-ecoflow"
)

// EcoFlowPowerOcean represents the EcoFlow PowerOcean meter
type EcoFlowPowerOcean struct {
	usage     string
	accessKey string
	secretKey string
	deviceId  string
	cache     time.Duration
	client    *ecoflow.Client
	dataG     func() (*ecoflow.GetCmdResponse, error)
}

func init() {
	registry.Add("ecoflow-powerocean", NewEcoFlowPowerOceanFromConfig)
}

// NewEcoFlowPowerOceanFromConfig creates an EcoFlow PowerOcean meter from generic config
func NewEcoFlowPowerOceanFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		AccessKey string
		SecretKey string
		DeviceId  string
		Usage     string
		Cache     time.Duration
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	if cc.AccessKey == "" {
		return nil, errors.New("missing access key")
	}
	if cc.SecretKey == "" {
		return nil, errors.New("missing secret key")
	}
	if cc.DeviceId == "" {
		return nil, errors.New("missing device ID")
	}
	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	m, err := NewEcoFlowPowerOcean(cc.AccessKey, cc.SecretKey, cc.DeviceId, cc.Usage, cc.Cache)
	if err != nil {
		return nil, err
	}
	// Optionally validate connection, but don't fail hard if it doesn't work
	if err := m.ValidateConnection(); err != nil {
		log := util.NewLogger("ecoflow-powerocean").Redact(cc.AccessKey, cc.SecretKey)
		log.DEBUG.Printf("connection test failed: %v", err)
	}

	return m, nil
}

// NewEcoFlowPowerOcean constructs the EcoFlowPowerOcean struct
func NewEcoFlowPowerOcean(accessKey, secretKey, deviceId, usage string, cache time.Duration) (*EcoFlowPowerOcean, error) {
	client := ecoflow.NewEcoflowClient(accessKey, secretKey)
	m := &EcoFlowPowerOcean{
		accessKey: accessKey,
		secretKey: secretKey,
		deviceId:  deviceId,
		usage:     usage,
		cache:     cache,
		client:    client,
	}
	switch usage {
	case "grid", "pv", "battery":
		m.dataG = util.Cached(m.getData, cache)
	default:
		return nil, fmt.Errorf("invalid usage: %s", usage)
	}
	return m, nil
}

// ValidateConnection calls one of the data methods to validate API connection
func (m *EcoFlowPowerOcean) ValidateConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := m.client.GetDeviceParameters(ctx, m.deviceId, []string{"sysGridPwr"})
	return err
}

// getData retrieves device parameters from EcoFlow API
func (m *EcoFlowPowerOcean) getData() (*ecoflow.GetCmdResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var params []string
	switch m.usage {
	case "grid":
		params = []string{"sysGridPwr"}
	case "pv":
		params = []string{"mpptHeartBeat"}
	case "battery":
		params = []string{"bpPwr", "bpSoc"}
	}

	return m.client.GetDeviceParameters(ctx, m.deviceId, params)
}

var _ api.Meter = (*EcoFlowPowerOcean)(nil)

// CurrentPower implements the api.Meter interface
func (m *EcoFlowPowerOcean) CurrentPower() (float64, error) {
	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	// sysGridPwr responds with int
	// mpptPwr responds with an array of floats
	// bpPwr responds with int
	// bpSoc responds with int
	switch m.usage {
	case "grid":
		return extractFloat(response.Data, "sysGridPwr")
	case "pv":
		return sumMpptPwrFloat(response.Data)
	case "battery":
		pwr, err := extractFloat(response.Data, "bpPwr")
		if err != nil {
			return 0, err
		}
		return -pwr, nil // invert battery power: ecoflow returns negative when discharging and positive when charging.
	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// convertToFloat64 attempts to convert an interface{} to float64, supporting int and float64 types.
func convertToFloat64(val interface{}) (float64, error) {
	if val == nil {
		return 0, errors.New("value is nil")
	}
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unsupported value type: %T", val)
	}
}

// extractFloat extracts a float64 or int value from a map by key.
func extractFloat(data map[string]interface{}, key string) (float64, error) {
	if data != nil {
		if v, ok := data[key]; ok {
			return convertToFloat64(v)
		}
	}
	return 0, fmt.Errorf("data not available for key: %s", key)
}

// sumMpptPwrFloat sums all 'pwr' float64 or int values in the mpptHeartBeat array.
func sumMpptPwrFloat(data map[string]interface{}) (float64, error) {
	arr, ok := data["mpptHeartBeat"].([]interface{})
	if !ok || len(arr) == 0 {
		return 0, fmt.Errorf("invalid mpptHeartBeat data structure")
	}
	var sum float64
	for _, item := range arr {
		mppt, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		mpptPvArr, ok := mppt["mpptPv"].([]interface{})
		if !ok {
			continue
		}
		for _, pv := range mpptPvArr {
			pvMap, ok := pv.(map[string]interface{})
			if !ok {
				continue
			}
			if pwrVal, ok := pvMap["pwr"]; ok {
				f, err := convertToFloat64(pwrVal)
				if err == nil {
					sum += f
				}
			}
		}
	}
	return sum, nil
}

// Soc implements the api.Battery interface for battery usage
func (m *EcoFlowPowerOcean) Soc() (float64, error) {
	if m.usage != "battery" {
		return 0, api.ErrNotAvailable
	}

	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	// Access the data from the GetCmdResponse
	if response.Data != nil {
		if soc, ok := response.Data["bpSoc"]; ok {
			return convertToFloat64(soc)
		}
	}

	return 0, errors.New("SOC data not available")
}
