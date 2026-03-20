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

// EcoFlowStream represents the EcoFlow Stream meter
type EcoFlowStream struct {
	ctx    context.Context
	usage  string
	serial string
	cache  time.Duration
	client *ecoflow.Client
	dataG  func() (*ecoflow.GetCmdResponse, error)
}

func init() {
	registry.AddCtx("ecoflow-stream", NewEcoFlowStreamFromConfig)
}

// NewEcoFlowStreamFromConfig creates an EcoFlow Stream meter from generic config
func NewEcoFlowStreamFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		AccessKey string
		SecretKey string
		Serial    string
		Usage     string
		Region    string
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
		return nil, errors.New("missing serial")
	}
	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}
	var baseUrl string
	switch cc.Region {
	case "":
		return nil, errors.New("missing region")
	case "auto":
		baseUrl = "https://api.ecoflow.com"
	case "europe":
		baseUrl = "https://api-e.ecoflow.com"
	case "america":
		baseUrl = "https://api-a.ecoflow.com"
	default:
		return nil, fmt.Errorf("invalid region: %s", cc.Region)
	}

	m, err := NewEcoFlowStream(ctx, cc.AccessKey, cc.SecretKey, cc.Serial, cc.Usage, baseUrl, cc.Cache)
	if err != nil {
		return nil, err
	}

	if cc.Usage == "battery" {
		return &EcoFlowStreamBattery{m}, nil
	}

	return m, nil
}

// NewEcoFlowStream constructs the EcoFlowStream struct
func NewEcoFlowStream(ctx context.Context, accessKey, secretKey, serial, usage, baseUrl string, cache time.Duration) (*EcoFlowStream, error) {
	client := ecoflow.NewEcoflowClient(accessKey, secretKey, ecoflow.WithBaseUrl(baseUrl))
	m := &EcoFlowStream{
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
func (m *EcoFlowStream) getData() (*ecoflow.GetCmdResponse, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	var params []string
	switch m.usage {
	case "grid":
		params = []string{"powGetSysGrid"}
	case "pv":
		params = []string{"powGetPvSum"}
	case "battery":
		params = []string{"powGetBpCms", "cmsBattSoc"}
	}

	return m.client.GetDeviceParameters(ctx, m.serial, params)
}

var _ api.Meter = (*EcoFlowStream)(nil)

// CurrentPower implements the api.Meter interface
func (m *EcoFlowStream) CurrentPower() (float64, error) {
	res, err := m.dataG()
	if err != nil {
		return 0, err
	}

	// powGetSysGrid responds with int
	// powGetPvSum responds with an array of floats
	// powGetBpCms responds with int
	// cmsBattSoc responds with int
	switch m.usage {
	case "grid":
		return extractFloat(res.Data, "powGetSysGrid")
	case "pv":
		return extractFloat(res.Data, "powGetPvSum")
	case "battery":
		pwr, err := extractFloat(res.Data, "powGetBpCms")
		if err != nil {
			return 0, err
		}
		return -pwr, nil // invert battery power: ecoflow returns negative when discharging and positive when charging.
	default:
		return 0, fmt.Errorf("invalid usage: %s", m.usage)
	}
}

// EcoFlowStreamBattery represents the EcoFlow Stream battery decorator
type EcoFlowStreamBattery struct {
	*EcoFlowStream
}

// Soc implements the api.Battery interface for battery usage
func (m *EcoFlowStreamBattery) Soc() (float64, error) {
	res, err := m.dataG()
	if err != nil {
		return 0, err
	}

	return extractFloat(res.Data, "cmsBattSoc")
}
