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

// EcoFlow represents the EcoFlow  meter
type EcoFlow struct {
	ctx    context.Context
	usage  string
	serial string
	cache  time.Duration
	client *ecoflow.Client
	dataG  func() (*ecoflow.GetCmdResponse, error)

	gridPower, pvPower, batteryPower, batterySoc string
}

func init() {
	registry.AddCtx("ecoflow", NewEcoFlowFromConfig)
}

// NewEcoFlowFromConfig creates an EcoFlow  meter from generic config
func NewEcoFlowFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		Usage                                        string
		AccessKey, SecretKey, Serial, Region         string
		GridPower, PvPower, BatteryPower, BatterySoc string
		Cache                                        time.Duration
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
		return nil, errors.New("missing serial")
	}
	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	var uri string
	switch cc.Region {
	case "auto":
		uri = "https://api.ecoflow.com"
	case "europe":
		uri = "https://api-e.ecoflow.com"
	case "america":
		uri = "https://api-a.ecoflow.com"
	default:
		return nil, fmt.Errorf("invalid region: %s", cc.Region)
	}

	m, err := NewEcoFlow(ctx, cc.AccessKey, cc.SecretKey, cc.Serial, cc.Usage, uri,
		cc.GridPower, cc.PvPower, cc.BatteryPower, cc.BatterySoc, cc.Cache)
	if err != nil {
		return nil, err
	}

	if cc.Usage == "battery" {
		return &EcoFlowBattery{m}, nil
	}

	return m, nil
}

// NewEcoFlow constructs the EcoFlow struct
func NewEcoFlow(ctx context.Context, accessKey, secretKey, serial, usage, uri string,
	gridPower, pvPower, batteryPower, batterySoc string, cache time.Duration) (*EcoFlow, error) {
	m := &EcoFlow{
		ctx:          ctx,
		serial:       serial,
		usage:        usage,
		cache:        cache,
		client:       ecoflow.NewEcoflowClient(accessKey, secretKey, ecoflow.WithBaseUrl(uri)),
		gridPower:    gridPower,
		pvPower:      pvPower,
		batteryPower: batteryPower,
		batterySoc:   batterySoc,
	}

	m.dataG = util.Cached(m.getData, cache)

	return m, nil
}

// getData retrieves device parameters from EcoFlow API
func (m *EcoFlow) getData() (*ecoflow.GetCmdResponse, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	var params []string
	switch m.usage {
	case "grid":
		params = []string{m.gridPower}
	case "pv":
		params = []string{m.pvPower}
	case "battery":
		params = []string{m.batteryPower, m.batterySoc}
	}

	return m.client.GetDeviceParameters(ctx, m.serial, params)
}

var _ api.Meter = (*EcoFlow)(nil)

// CurrentPower implements the api.Meter interface
func (m *EcoFlow) CurrentPower() (float64, error) {
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
		return ecoflowValue(response.Data, m.gridPower)
	case "pv":
		return ecoflowValue(response.Data, m.pvPower)
	case "battery":
		pwr, err := ecoflowValue(response.Data, m.batteryPower)
		if err != nil {
			return 0, err
		}
		return -pwr, nil // invert battery power: ecoflow returns negative when discharging and positive when charging.
	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// extractFloat extracts a float64 or int value from a map by key.
func ecoflowValue(data map[string]any, key string) (float64, error) {
	if data != nil {
		if v, ok := data[key]; ok {
			return cast.ToFloat64E(v)
		}
	}
	return 0, api.ErrNotAvailable
}

// EcoFlowBattery represents the EcoFlow  battery decorator
type EcoFlowBattery struct {
	*EcoFlow
}

// Soc implements the api.Battery interface for battery usage
func (m *EcoFlowBattery) Soc() (float64, error) {
	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	return ecoflowValue(response.Data, m.batterySoc)
}
