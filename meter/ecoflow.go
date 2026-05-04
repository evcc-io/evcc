package meter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/spf13/cast"
	"github.com/tess1o/go-ecoflow"
)

// EcoFlow represents the EcoFlow  meter
type EcoFlow struct {
	usage  string
	serial string
	cache  time.Duration
	client *ecoflow.Client
	dataG  func() (*ecoflow.GetCmdResponse, error)

	power, batterySoc string
}

func init() {
	registry.Add("ecoflow", NewEcoFlowFromConfig)
}

// NewEcoFlowFromConfig creates an EcoFlow  meter from generic config
func NewEcoFlowFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		batteryCapacity                      `mapstructure:",squash"`
		batteryPowerLimits                   `mapstructure:",squash"`
		batterySocLimits                     `mapstructure:",squash"`
		Usage                                string
		AccessKey, SecretKey, Serial, Region string
		Power, Soc                           string
		Cache                                time.Duration
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

	m, err := NewEcoFlow(cc.AccessKey, cc.SecretKey, cc.Serial, cc.Usage, uri, cc.Power, cc.Soc, cc.Cache)
	if err != nil {
		return nil, err
	}

	if cc.Usage == "battery" {
		return decorateMeterBattery(
			m, nil, m.soc, cc.batteryCapacity.Decorator(),
			cc.batterySocLimits.Decorator(), cc.batteryPowerLimits.Decorator(), nil,
		), nil
	}

	return m, nil
}

// NewEcoFlow constructs the EcoFlow struct
func NewEcoFlow(accessKey, secretKey, serial, usage, uri string,
	power, soc string, cache time.Duration) (*EcoFlow, error) {
	log := util.NewLogger("ecoflow").Redact(accessKey, secretKey, serial)

	m := &EcoFlow{
		serial: serial,
		usage:  usage,
		cache:  cache,
		client: ecoflow.NewEcoflowClient(accessKey, secretKey,
			ecoflow.WithBaseUrl(uri),
			ecoflow.WithHttpClient(request.NewClient(log)),
		),
		power:      power,
		batterySoc: soc,
	}

	m.dataG = util.Cached(m.getData, cache)

	return m, nil
}

// getData retrieves device parameters from EcoFlow API
func (m *EcoFlow) getData() (*ecoflow.GetCmdResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := []string{m.power}

	if m.usage == "battery" {
		params = append(params, m.batterySoc)
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

	pwr, err := ecoflowValue(response.Data, m.power)
	if err != nil {
		return 0, err
	}

	if m.usage == "battery" {
		pwr = -pwr // invert battery power: ecoflow returns negative when discharging and positive when charging.
	}

	return pwr, nil
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

// soc returns the battery state of charge
func (m *EcoFlow) soc() (float64, error) {
	response, err := m.dataG()
	if err != nil {
		return 0, err
	}

	return ecoflowValue(response.Data, m.batterySoc)
}
