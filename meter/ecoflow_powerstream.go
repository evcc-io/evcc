package meter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// https://developer-eu.ecoflow.com/us/document/wn511

// EcoFlowPowerStreamData represents the full device status from PowerStream API
type EcoFlowPowerStreamData struct {
	// Power values (from heartbeat)
	Pv1InputWatts  float64 `json:"pv1InputWatts"`  // PV1 input power (W)
	Pv2InputWatts  float64 `json:"pv2InputWatts"`  // PV2 input power (W)
	BatInputWatts  float64 `json:"batInputWatts"`  // Battery input/output power (W), positive=discharge, negative=charge
	BatInputCur    int     `json:"batInputCur"`    // Battery current (0.1A), positive=discharge, negative=charge
	InvOutputWatts float64 `json:"invOutputWatts"` // Inverter AC output power (W)
	BatSoc         int     `json:"batSoc"`         // Battery SOC (%)
	SupplyPriority int     `json:"supplyPriority"` // Power supply priority (0=supply, 1=storage)
	InvOnOff       int     `json:"invOnOff"`       // Inverter on/off (0=off, 1=on)
	PermanentWatts int     `json:"permanentWatts"` // Custom load power (W)
	LowerLimit     int     `json:"lowerLimit"`     // Battery discharge limit (%)
	UpperLimit     int     `json:"upperLimit"`     // Battery charge limit (%)
	FeedProtect    int     `json:"feedProtect"`    // Feed-in protection (0=off, 1=on)
}

// EcoFlowPowerStream represents an EcoFlow PowerStream Micro-Inverter (Inverter + Battery)
type EcoFlowPowerStream struct {
	*request.Helper
	log       *util.Logger
	uri       string
	sn        string // device serial number
	accessKey string // API access key
	secretKey string // API secret key for signing
	usage     string // pv, grid, battery
	cache     time.Duration
	dataG     util.Cacheable[EcoFlowPowerStreamData]
}

func init() {
	registry.AddCtx("ecoflow-powerstream", NewEcoFlowPowerStreamFromConfig)
}

// NewEcoFlowPowerStreamFromConfig creates EcoFlow PowerStream meter from config
func NewEcoFlowPowerStreamFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
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
		return nil, fmt.Errorf("ecoflow-powerstream: missing uri, sn, accessKey or secretKey")
	}

	device, err := NewEcoFlowPowerStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Usage, cc.Cache)
	if err != nil {
		return nil, err
	}

	// For battery usage, wrap in battery interface
	if cc.Usage == "battery" {
		return &EcoFlowPowerStreamBattery{device}, nil
	}

	// For other usages (pv, grid), return as meter
	return device, nil
}

// NewEcoFlowPowerStream creates an EcoFlow PowerStream device for use as a meter
func NewEcoFlowPowerStream(uri, sn, accessKey, secretKey, usage string, cache time.Duration) (*EcoFlowPowerStream, error) {
	if uri == "" || sn == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("ecoflow-powerstream: missing uri, sn, accessKey or secretKey")
	}

	log := util.NewLogger("ecoflow-powerstream").Redact(accessKey, secretKey)

	device := &EcoFlowPowerStream{
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
	device.Client.Transport = charger.NewEcoFlowAuthTransport(accessKey, secretKey)

	// Create cached data fetcher
	device.dataG = util.ResettableCached(device.getQuotaAll, cache)

	return device, nil
}

// getQuotaAll fetches device quota data from API
func (d *EcoFlowPowerStream) getQuotaAll() (EcoFlowPowerStreamData, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.uri, d.sn)

	type response struct {
		Code    string                  `json:"code"`
		Message string                  `json:"message"`
		Data    EcoFlowPowerStreamData `json:"data"`
	}

	var res response
	err := d.GetJSON(uri, &res)
	if err != nil {
		return EcoFlowPowerStreamData{}, err
	}

	if res.Code != "0" {
		return EcoFlowPowerStreamData{}, fmt.Errorf("API error: %s", res.Message)
	}

	return res.Data, nil
}

// CurrentPower implements api.Meter
func (d *EcoFlowPowerStream) CurrentPower() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch d.usage {
	case "pv":
		// PV power is sum of both strings
		return data.Pv1InputWatts + data.Pv2InputWatts, nil
	case "battery":
		// Battery power: positive=discharge, negative=charge
		return data.BatInputWatts, nil
	case "grid":
		// Grid power (calculated from AC output)
		return data.InvOutputWatts, nil
	default:
		return 0, fmt.Errorf("unknown usage type: %s", d.usage)
	}
}

// EcoFlowPowerStreamBattery wraps EcoFlowPowerStream for battery interface
type EcoFlowPowerStreamBattery struct {
	*EcoFlowPowerStream
}

// Soc implements api.Battery
func (d *EcoFlowPowerStreamBattery) Soc() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	return float64(data.BatSoc), nil
}

var _ api.Meter = (*EcoFlowPowerStream)(nil)
var _ api.Meter = (*EcoFlowPowerStreamBattery)(nil)
var _ api.Battery = (*EcoFlowPowerStreamBattery)(nil)
