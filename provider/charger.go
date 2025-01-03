package provider

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	charger "github.com/evcc-io/evcc/charger/config"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

type chargerProvider struct {
	charger api.Charger
}

func init() {
	registry.AddCtx("charger", NewChargerEnableFromConfig)
}

// NewChargerEnableFromConfig creates type conversion provider
func NewChargerEnableFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Charger config.Typed
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Charger.Type == "" {
		return nil, fmt.Errorf("missing charger")
	}

	charger, err := charger.NewFromConfig(ctx, cc.Charger.Type, cc.Charger.Other)
	if err != nil {
		return nil, err
	}

	o := &chargerProvider{
		charger: charger,
	}

	return o, nil
}

var _ BoolProvider = (*chargerProvider)(nil)

func (o *chargerProvider) BoolGetter() (func() (bool, error), error) {
	return func() (bool, error) {
		return o.charger.Enabled()
	}, nil
}

var _ SetBoolProvider = (*chargerProvider)(nil)

func (o *chargerProvider) BoolSetter(param string) (func(bool) error, error) {
	return func(val bool) error {
		return o.charger.Enable(val)
	}, nil
}
