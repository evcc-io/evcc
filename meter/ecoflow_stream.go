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
	registry.AddCtx("ecoflow-stream-pv", NewEcoflowStreamPVFromConfig)
	registry.AddCtx("ecoflow-stream-grid", NewEcoflowStreamGridFromConfig)
	// Battery implements the Meter interface, so we can register it as a meter
	registry.AddCtx("ecoflow-stream-battery", func(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
		bat, err := NewEcoflowStreamBatteryFromConfig(ctx, other)
		return bat.(api.Meter), err
	})
}

// NewEcoflowStreamPVFromConfig creates EcoFlow Stream PV generation meter
func NewEcoflowStreamPVFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI   string        `mapstructure:"uri"`
		SN    string        `mapstructure:"sn"`
		Token string        `mapstructure:"token"`
		Cache time.Duration `mapstructure:"cache"`
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.SN == "" || cc.Token == "" {
		return nil, fmt.Errorf("ecoflow-stream-pv: missing uri, sn or token")
	}

	parent, err := charger.NewEcoflowStream(cc.URI, cc.SN, cc.Token, cc.Cache)
	if err != nil {
		return nil, err
	}

	return charger.NewEcoflowStreamPV(parent.(*charger.EcoflowStream), cc.Cache), nil
}

// NewEcoflowStreamGridFromConfig creates EcoFlow Stream grid meter
func NewEcoflowStreamGridFromConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI   string        `mapstructure:"uri"`
		SN    string        `mapstructure:"sn"`
		Token string        `mapstructure:"token"`
		Cache time.Duration `mapstructure:"cache"`
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.SN == "" || cc.Token == "" {
		return nil, fmt.Errorf("ecoflow-stream-grid: missing uri, sn or token")
	}

	parent, err := charger.NewEcoflowStream(cc.URI, cc.SN, cc.Token, cc.Cache)
	if err != nil {
		return nil, err
	}

	return charger.NewEcoflowStreamGrid(parent.(*charger.EcoflowStream), cc.Cache), nil
}

// NewEcoflowStreamBatteryFromConfig creates EcoFlow Stream battery meter
func NewEcoflowStreamBatteryFromConfig(ctx context.Context, other map[string]interface{}) (api.Battery, error) {
	cc := struct {
		URI   string        `mapstructure:"uri"`
		SN    string        `mapstructure:"sn"`
		Token string        `mapstructure:"token"`
		Cache time.Duration `mapstructure:"cache"`
	}{
		Cache: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.SN == "" || cc.Token == "" {
		return nil, fmt.Errorf("ecoflow-stream-battery: missing uri, sn or token")
	}

	parent, err := charger.NewEcoflowStream(cc.URI, cc.SN, cc.Token, cc.Cache)
	if err != nil {
		return nil, err
	}

	return charger.NewEcoflowStreamBattery(parent.(*charger.EcoflowStream), cc.Cache), nil
}
