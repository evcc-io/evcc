package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("valid", NewValidFromConfig)
}

// validPlugin validates a reading via a second reading
type validPlugin struct {
	ctx   context.Context
	valid func() (bool, error)
	value Config
}

// NewValidFromConfig creates valid provider
func NewValidFromConfig(ctx context.Context, other map[string]interface{}) (Plugin, error) {
	var cc struct {
		Valid, Value Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	valid, err := cc.Valid.BoolGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("valid: %w", err)
	}

	o := NewValidPlugin(ctx, valid, cc.Value)

	return o, nil
}

// NewValidPlugin creates valid provider
func NewValidPlugin(ctx context.Context, valid func() (bool, error), value Config) *validPlugin {
	return &validPlugin{
		ctx:   ctx,
		valid: valid,
		value: value,
	}
}

var _ StringGetter = (*validPlugin)(nil)

func validGetter[T any](o *validPlugin, valuer func(ctx context.Context) (func() (T, error), error)) (func() (T, error), error) {
	value, err := valuer(o.ctx)
	if err != nil {
		return nil, fmt.Errorf("valid: %w", err)
	}

	return func() (T, error) {
		var zero T

		valid, err := o.valid()
		if err != nil {
			return zero, err
		}
		if !valid {
			return zero, errors.New("invalid")
		}

		return value()
	}, nil
}

var _ StringGetter = (*validPlugin)(nil)

func (o *validPlugin) StringGetter() (func() (string, error), error) {
	return validGetter(o, o.value.StringGetter)
}

var _ FloatGetter = (*validPlugin)(nil)

func (o *validPlugin) FloatGetter() (func() (float64, error), error) {
	return validGetter(o, o.value.FloatGetter)
}

var _ IntGetter = (*validPlugin)(nil)

func (o *validPlugin) IntGetter() (func() (int64, error), error) {
	return validGetter(o, o.value.IntGetter)
}
