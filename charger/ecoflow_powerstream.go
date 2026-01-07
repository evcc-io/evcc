package charger

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// https://developer-eu.ecoflow.com/us/document/wn511

// EcoFlowPowerStream represents an EcoFlow PowerStream micro-inverter
type EcoFlowPowerStream struct {
	*request.Helper
	log       *util.Logger
	uri       string
	sn        string // device serial number
	accessKey string // API access key
	secretKey string // API secret key for signing
	usage     string // charger, pv, grid, battery
	cache     time.Duration
	dataG     util.Cacheable[QuotaAllDataPowerStream]
}

// QuotaAllDataPowerStream represents the full device status from PowerStream API
type QuotaAllDataPowerStream struct {
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

func init() {
	registry.AddCtx("ecoflow-powerstream", NewEcoFlowPowerStreamFromConfig)
}

// NewEcoFlowPowerStream creates an EcoFlow PowerStream device
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
	device.Client.Transport = &authTransportPowerStream{
		base:      transport.Default(),
		accessKey: accessKey,
		secretKey: secretKey,
	}

	// Create cached data fetcher
	device.dataG = util.ResettableCached(device.getQuotaAll, cache)

	return device, nil
}

// NewEcoFlowPowerStreamFromConfig creates EcoFlow PowerStream device from config
func NewEcoFlowPowerStreamFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI       string        `mapstructure:"uri"`
		SN        string        `mapstructure:"sn"`
		AccessKey string        `mapstructure:"accessKey"`
		SecretKey string        `mapstructure:"secretKey"`
		Usage     string        `mapstructure:"usage"`
		Cache     time.Duration `mapstructure:"cache"`
	}{
		Usage: "charger",
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

	return device, nil
}

// getQuotaAll fetches device quota data from API
func (d *EcoFlowPowerStream) getQuotaAll() (QuotaAllDataPowerStream, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.uri, d.sn)

	type response struct {
		Code    string                  `json:"code"`
		Message string                  `json:"message"`
		Data    QuotaAllDataPowerStream `json:"data"`
	}

	var res response
	err := d.GetJSON(uri, &res)
	if err != nil {
		return QuotaAllDataPowerStream{}, err
	}

	if res.Code != "0" {
		return QuotaAllDataPowerStream{}, fmt.Errorf("API error: %s", res.Message)
	}

	return res.Data, nil
}

// Status implements api.Charger
func (d *EcoFlowPowerStream) Status() (api.ChargeStatus, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	if d.usage != "charger" {
		return api.StatusNone, fmt.Errorf("status not available for usage type %s", d.usage)
	}

	// Check if inverter is active and supplying power
	if data.InvOutputWatts > 100 {
		return api.StatusC, nil // Charging/supplying
	}

	return api.StatusB, nil
}

// Enabled implements api.Charger
func (d *EcoFlowPowerStream) Enabled() (bool, error) {
	if d.usage != "charger" {
		return false, fmt.Errorf("enabled not available for usage type %s", d.usage)
	}

	data, err := d.dataG.Get()
	if err != nil {
		return false, err
	}

	return data.InvOnOff == 1, nil
}

// Enable implements api.Charger
func (d *EcoFlowPowerStream) Enable(enable bool) error {
	if d.usage != "charger" {
		return fmt.Errorf("enable not available for usage type %s", d.usage)
	}

	// Inverter control not yet implemented - device is always on
	if !enable {
		return fmt.Errorf("inverter disable not supported")
	}
	return nil
}

// MaxCurrent implements api.Charger
func (d *EcoFlowPowerStream) MaxCurrent(current int64) error {
	if d.usage != "charger" {
		return fmt.Errorf("maxcurrent not available for usage type %s", d.usage)
	}

	// TODO: Set permanent watts (custom load power)
	return fmt.Errorf("max current control not yet implemented")
}

// CurrentPower implements api.Meter
func (d *EcoFlowPowerStream) CurrentPower() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch d.usage {
	case "charger":
		// Charger returns inverter output power (AC side)
		return data.InvOutputWatts, nil
	case "pv":
		// PV power is sum of both strings
		return data.Pv1InputWatts + data.Pv2InputWatts, nil
	case "battery":
		// Battery power: positive=discharge, negative=charge
		return data.BatInputWatts, nil
	case "grid":
		// Grid power would be calculated from AC output
		return data.InvOutputWatts, nil
	default:
		return 0, fmt.Errorf("unknown usage type: %s", d.usage)
	}
}

// Soc implements api.Battery
func (d *EcoFlowPowerStream) Soc() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	return float64(data.BatSoc), nil
}

var _ api.Charger = (*EcoFlowPowerStream)(nil)
var _ api.Meter = (*EcoFlowPowerStream)(nil)
var _ api.Battery = (*EcoFlowPowerStream)(nil)

// authTransportPowerStream adds HMAC-SHA256 signed authentication headers to requests
type authTransportPowerStream struct {
	base      http.RoundTripper
	accessKey string
	secretKey string
}

func (t *authTransportPowerStream) RoundTrip(req *http.Request) (*http.Response, error) {
	nonce := generateNonce()
	timestamp := time.Now().UnixMilli()

	var signStr string
	if req.URL.RawQuery != "" {
		signStr = req.URL.RawQuery
	}

	if signStr != "" {
		signStr += "&"
	}
	signStr += fmt.Sprintf("accessKey=%s&nonce=%d&timestamp=%d", t.accessKey, nonce, timestamp)

	sign := hmacSHA256(signStr, t.secretKey)

	req.Header.Set("accessKey", t.accessKey)
	req.Header.Set("nonce", fmt.Sprintf("%d", nonce))
	req.Header.Set("timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("sign", sign)

	return t.base.RoundTrip(req)
}
