package meter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// EcoFlowStream represents an EcoFlow Stream Energy Management System (Inverter + Battery)
type EcoFlowStream struct {
	*request.Helper
	log       *util.Logger
	uri       string
	sn        string // device serial number
	accessKey string // API access key
	secretKey string // API secret key for signing
	usage     string // pv, grid, battery
	cache     time.Duration
	dataG     util.Cacheable[EcoFlowStreamData]
}

func init() {
	registry.AddCtx("ecoflow-stream", NewEcoFlowStreamFromConfig)
}

// NewEcoFlowStreamFromConfig creates EcoFlow Stream meter from config
func NewEcoFlowStreamFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI       string        `mapstructure:"uri"`
		SN        string        `mapstructure:"sn"`
		AccessKey string        `mapstructure:"accessKey"`
		SecretKey string        `mapstructure:"secretKey"`
		Usage     string        `mapstructure:"usage"`
		Cache     time.Duration `mapstructure:"cache"`
	}{
		Usage: "grid",
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.SN == "" || cc.AccessKey == "" || cc.SecretKey == "" {
		return nil, fmt.Errorf("ecoflow-stream: missing uri, sn, accessKey or secretKey")
	}

	device, err := NewEcoFlowStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Usage, cc.Cache)
	if err != nil {
		return nil, err
	}

	// For battery usage, wrap in battery interface
	if cc.Usage == "battery" {
		return &EcoFlowStreamBattery{device}, nil
	}

	// For other usages (pv, grid), return as meter
	return device, nil
}

// NewEcoFlowStream creates an EcoFlow Stream device for use as a meter
func NewEcoFlowStream(uri, sn, accessKey, secretKey, usage string, cache time.Duration) (*EcoFlowStream, error) {
	if uri == "" || sn == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("ecoflow-stream: missing uri, sn, accessKey or secretKey")
	}

	log := util.NewLogger("ecoflow-stream").Redact(accessKey, secretKey)

	device := &EcoFlowStream{
		Helper:    request.NewHelper(log),
		log:       log,
		uri:       strings.TrimSuffix(uri, "/"),
		sn:        sn,
		accessKey: accessKey,
		secretKey: secretKey,
		usage:     strings.ToLower(usage),
		cache:     cache,
	}

	// Set authorization header using custom transport with HMAC-SHA256 signature
	device.Client.Transport = NewEcoFlowAuthTransport(accessKey, secretKey)

	// Create cached data fetcher
	device.dataG = util.ResettableCached(device.getQuotaAll, cache)

	return device, nil
}

// getQuotaAll fetches device quota data from API
func (d *EcoFlowStream) getQuotaAll() (EcoFlowStreamData, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.uri, d.sn)

	var res ecoflowResponse[EcoFlowStreamData]
	err := d.GetJSON(uri, &res)
	if err != nil {
		return EcoFlowStreamData{}, err
	}

	if res.Code != "0" {
		return EcoFlowStreamData{}, fmt.Errorf("API error: %s", res.Message)
	}

	return res.Data, nil
}

// CurrentPower implements api.Meter
func (d *EcoFlowStream) CurrentPower() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch d.usage {
	case "battery":
		// Battery: negative = charging, positive = discharging
		// EcoFlow convention: PowGetBpCms positive when discharging, negative when charging
		// evcc convention: negative when discharging, positive when charging
		return -data.PowGetBpCms, nil
	case "pv":
		return data.PowGetPvSum, nil
	case "grid":
		return data.PowGetSysGrid, nil
	default:
		return 0, fmt.Errorf("unknown usage type: %s", d.usage)
	}
}

// EcoFlowStreamBattery wraps EcoFlowStream for battery interface
type EcoFlowStreamBattery struct {
	*EcoFlowStream
}

// Soc implements api.Battery
func (d *EcoFlowStreamBattery) Soc() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	return data.CmsBattSoc, nil
}

var _ api.Meter = (*EcoFlowStream)(nil)
var _ api.Meter = (*EcoFlowStreamBattery)(nil)
var _ api.Battery = (*EcoFlowStreamBattery)(nil)
