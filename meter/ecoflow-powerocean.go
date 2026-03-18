package meter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cast"
	"github.com/tess1o/go-ecoflow"
)

// EcoFlowPowerOcean represents the EcoFlow PowerOcean meter
type EcoFlowPowerOcean struct {
	ctx    context.Context
	usage  string
	serial string
	cache  time.Duration
	client *ecoflow.Client
	dataG  func() (*ecoflow.GetCmdResponse, error)
}

func init() {
	registry.AddCtx("ecoflow-powerocean", NewEcoFlowPowerOceanFromConfig)
}

// NewEcoFlowPowerOceanFromConfig creates an EcoFlow PowerOcean meter from generic config
func NewEcoFlowPowerOceanFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		AccessKey string
		SecretKey string
		Serial    string
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
	if cc.Serial == "" {
		return nil, errors.New("missing serial number")
	}
	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	m, err := NewEcoFlowPowerOcean(ctx, cc.AccessKey, cc.SecretKey, cc.Serial, cc.Usage, cc.Cache)
	if err != nil {
		return nil, err
	}

	if cc.Usage == "battery" {
		return &EcoFlowPowerOceanBattery{m}, nil
	}

	return m, nil
}

// NewEcoFlowPowerOcean constructs the EcoFlowPowerOcean struct
func NewEcoFlowPowerOcean(ctx context.Context, accessKey, secretKey, serial, usage string, cache time.Duration) (*EcoFlowPowerOcean, error) {
	client := ecoflow.NewEcoflowClient(accessKey, secretKey)
	m := &EcoFlowPowerOcean{
		ctx:    ctx,
		serial: serial,
		usage:  usage,
		cache:  cache,
		client: client,
	}
	m.dataG = util.Cached(m.getData, cache)
	return m, nil
}

// getData retrieves device parameters from EcoFlow API
func (m *EcoFlowPowerOcean) getData() (*ecoflow.GetCmdResponse, error) {
	timedCtx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	var params []string
	switch m.usage {
	case "grid":
		params = []string{"sysGridPwr"}
	case "pv":
		params = []string{"mpptPwr"}
	case "battery":
		params = []string{"bpPwr", "bpSoc"}
	}

	return m.client.GetDeviceParameters(timedCtx, m.serial, params)
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
		return extractFloat(response.Data, "mpptPwr")
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

// extractFloat extracts a float64 or int value from a map by key.
func extractFloat(data map[string]any, key string) (float64, error) {
	if data != nil {
		if v, ok := data[key]; ok {
			return cast.ToFloat64E(v)
		}
	}
	return 0, fmt.Errorf("data not available for key: %s", key)
}

// EcoFlowPowerOceanBattery represents the EcoFlow PowerOcean battery decorator
type EcoFlowPowerOceanBattery struct {
	*EcoFlowPowerOcean
}

// Soc implements the api.Battery interface for battery usage
func (m *EcoFlowPowerOceanBattery) Soc() (float64, error) {
	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	return extractFloat(response.Data, "bpSoc")
}
