package ecoflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Stream represents an EcoFlow Stream Energy Management System (Inverter + Battery)
type Stream struct {
	*request.Helper
	log       *util.Logger
	uri       string
	sn        string // device serial number
	accessKey string // API access key
	secretKey string // API secret key for signing
	usage     string // pv, grid, battery
	cache     time.Duration
	dataG     util.Cacheable[StreamData]
}

var _ api.Meter = (*Stream)(nil)

// NewStreamFromConfig creates EcoFlow Stream meter from config
func NewStreamFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := &Config{}
	if err := cc.Decode(other, "ecoflow-stream"); err != nil {
		return nil, err
	}

	device, err := NewStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Usage, cc.Cache)
	if err != nil {
		return nil, err
	}

	// For battery usage, wrap in battery interface
	if cc.Usage == UsageBattery {
		return &StreamBattery{device}, nil
	}

	// For other usages (pv, grid), return as meter
	return device, nil
}

// NewStream creates an EcoFlow Stream device for use as a meter
func NewStream(uri, sn, accessKey, secretKey, usage string, cache time.Duration) (*Stream, error) {
	if uri == "" || sn == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("ecoflow-stream: missing uri, sn, accessKey or secretKey")
	}

	log := util.NewLogger("ecoflow-stream").Redact(accessKey, secretKey)

	device := &Stream{
		Helper:    request.NewHelper(log),
		log:       log,
		uri:       strings.TrimSuffix(uri, "/"),
		sn:        sn,
		accessKey: accessKey,
		secretKey: secretKey,
		usage:     usage,
		cache:     cache,
	}

	// Set authorization header using custom transport with HMAC-SHA256 signature,
	// wrapping the existing transport to preserve proxy/TLS/custom settings
	device.Client.Transport = NewEcoFlowAuthTransport(device.Client.Transport, accessKey, secretKey)

	// Create cached data fetcher
	device.dataG = util.ResettableCached(device.getQuotaAll, cache)

	return device, nil
}

// getQuotaAll fetches device quota data from API
func (d *Stream) getQuotaAll() (StreamData, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.uri, d.sn)

	var res ecoflowResponse[StreamData]
	if err := d.GetJSON(uri, &res); err != nil {
		return StreamData{}, err
	}

	if res.Code != "0" {
		return StreamData{}, fmt.Errorf("API error: %s", res.Message)
	}

	return res.Data, nil
}

// CurrentPower implements api.Meter
func (d *Stream) CurrentPower() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch d.usage {
	case UsageBattery:
		// Battery: negative = charging, positive = discharging
		// EcoFlow convention: PowGetBpCms positive when discharging, negative when charging
		// evcc convention: negative when discharging, positive when charging
		return -data.PowGetBpCms, nil
	case UsagePV:
		return data.PowGetPvSum, nil
	case UsageGrid:
		return data.PowGetSysGrid, nil
	default:
		return 0, fmt.Errorf("unknown usage type: %s", d.usage)
	}
}

// StreamBattery wraps Stream for battery interface
type StreamBattery struct {
	*Stream
}

var (
	_ api.Meter   = (*StreamBattery)(nil)
	_ api.Battery = (*StreamBattery)(nil)
)

// Soc implements the api.Battery interface
func (d *StreamBattery) Soc() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	return data.CmsBattSoc, nil
}
