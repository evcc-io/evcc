package charger

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// https://developer-eu.ecoflow.com/us/document/bkw

// EcoFlowStream represents an EcoFlow Stream series device
type EcoFlowStream struct {
	*request.Helper
	log       *util.Logger
	uri       string
	sn        string // device serial number
	accessKey string // API access key
	secretKey string // API secret key for signing
	usage     string // charger, relay1, relay2, pv, grid, battery
	cache     time.Duration
	dataG     util.Cacheable[QuotaAllData]
}

// QuotaAllData represents the full device status from API
type QuotaAllData struct {
	Relay2Onoff               bool            `json:"relay2Onoff"`         // AC1 switch (false=off, true=on)
	Relay3Onoff               bool            `json:"relay3Onoff"`         // AC2 switch (false=off, true=on)
	PowGetPvSum               float64         `json:"powGetPvSum"`         // Real-time PV power (W)
	FeedGridMode              int             `json:"feedGridMode"`        // Feed-in control (1-off, 2-on)
	GridConnectionPower       float64         `json:"gridConnectionPower"` // Grid port power (W)
	PowGetSysGrid             float64         `json:"powGetSysGrid"`       // System real-time grid power (W)
	PowGetSysLoad             float64         `json:"powGetSysLoad"`       // System real-time load power (W)
	CmsBattSoc                float64         `json:"cmsBattSoc"`          // Battery SOC (%)
	PowGetBpCms               float64         `json:"powGetBpCms"`         // Real-time battery power (W)
	BackupReverseSoc          int             `json:"backupReverseSoc"`    // Backup reserve level (%)
	CmsMaxChgSoc              int             `json:"cmsMaxChgSoc"`        // Charge limit (%)
	CmsMinDsgSoc              int             `json:"cmsMinDsgSoc"`        // Discharge limit (%)
	EnergyStrategyOperateMode map[string]bool `json:"energyStrategyOperateMode"`
	QuotaCloudTs              string          `json:"quota_cloud_ts"`
}

func init() {
	registry.AddCtx("ecoflow-stream", NewEcoFlowStreamFromConfig)
}

// NewEcoFlowStream creates an EcoFlow Stream device for use by meters
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

// NewEcoFlowStreamFromConfig creates EcoFlow Stream device from config
func NewEcoFlowStreamFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
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
		return nil, fmt.Errorf("ecoflow-stream: missing uri, sn, accessKey or secretKey")
	}

	device, err := NewEcoFlowStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Usage, cc.Cache)
	if err != nil {
		return nil, err
	}

	return device, nil
}

// getQuotaAll fetches device quota data from API
func (d *EcoFlowStream) getQuotaAll() (QuotaAllData, error) {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.uri, d.sn)

	type response struct {
		Code    string       `json:"code"`
		Message string       `json:"message"`
		Data    QuotaAllData `json:"data"`
	}

	var res response
	err := d.GetJSON(uri, &res)
	if err != nil {
		return QuotaAllData{}, err
	}

	if res.Code != "0" {
		return QuotaAllData{}, fmt.Errorf("API error: %s", res.Message)
	}

	return res.Data, nil
}

// Status implements api.Charger
func (d *EcoFlowStream) Status() (api.ChargeStatus, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return api.StatusA, err
	}

	// For non-charger usage (meter mode), always return StatusB
	if d.usage != "charger" {
		return api.StatusB, nil
	}

	// Check if actively charging (battery power > threshold)
	if data.PowGetBpCms > 100 {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

// Enabled implements api.Charger
func (d *EcoFlowStream) Enabled() (bool, error) {
	if d.usage != "charger" {
		return true, nil // Meters are always "enabled"
	}

	// For now, assume always enabled (relay control disabled)
	return true, nil
}

// Enable implements api.Charger
func (d *EcoFlowStream) Enable(enable bool) error {
	if d.usage != "charger" {
		return fmt.Errorf("enable not available for usage type %s", d.usage)
	}

	// Charger control not yet implemented - device is always on
	if !enable {
		return fmt.Errorf("charger disable not supported")
	}
	return nil
}

// MaxCurrent implements api.Charger
func (d *EcoFlowStream) MaxCurrent(current int64) error {
	return fmt.Errorf("not supported")
}

// CurrentPower implements api.Meter
func (d *EcoFlowStream) CurrentPower() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	switch d.usage {
	case "charger":
		// Charger uses system load power (not the relay)
		return data.PowGetSysLoad, nil
	case "relay1", "relay2":
		// Relays don't have individual power measurement, return 0
		return 0, nil
	case "battery":
		return -data.PowGetBpCms, nil // EcoFlow reports battery power with opposite sign; invert to match evcc convention
	case "pv":
		return data.PowGetPvSum, nil // PV generation power
	case "grid":
		return data.PowGetSysGrid, nil // Grid power
	default:
		return 0, fmt.Errorf("unknown usage type: %s", d.usage)
	}
}

// Soc implements api.Battery
func (d *EcoFlowStream) Soc() (float64, error) {
	data, err := d.dataG.Get()
	if err != nil {
		return 0, err
	}

	return data.CmsBattSoc, nil
}

var _ api.Charger = (*EcoFlowStream)(nil)
var _ api.Meter = (*EcoFlowStream)(nil)
var _ api.Battery = (*EcoFlowStream)(nil)
