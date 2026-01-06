package meter

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/util"
)

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

	// Create device with specified usage type
	device, err := charger.NewEcoFlowStream(cc.URI, cc.SN, cc.AccessKey, cc.SecretKey, cc.Usage, cc.Cache)
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

// EcoFlowStreamBattery wraps EcoFlowStream for battery interface
type EcoFlowStreamBattery struct {
	*charger.EcoFlowStream
}

var _ api.Meter = (*EcoFlowStreamBattery)(nil)
var _ api.Battery = (*EcoFlowStreamBattery)(nil)
