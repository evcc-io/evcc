package plugin

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
)

type constPlugin struct {
	ctx context.Context
	str string
	set Config
}

func init() {
	registry.AddCtx("const", NewConstFromConfig)
}

// NewConstFromConfig creates const provider
func NewConstFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Value             string
		pipeline.Settings `mapstructure:",squash"`
		Set               Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p := &constPlugin{
		ctx: ctx,
		str: cc.Value,
		set: cc.Set,
	}

	return p, nil
}

var _ StringGetter = (*constPlugin)(nil)

func (p *constPlugin) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		return p.str, nil
	}, nil
}

var _ IntGetter = (*constPlugin)(nil)

func (p *constPlugin) IntGetter() (func() (int64, error), error) {
	val, err := strconv.ParseInt(p.str, 10, 64)
	if err != nil && p.str == "" {
		err = nil
	}

	return func() (int64, error) {
		return val, err
	}, err
}

var _ FloatGetter = (*constPlugin)(nil)

func (p *constPlugin) FloatGetter() (func() (float64, error), error) {
	val, err := strconv.ParseFloat(p.str, 64)
	if err != nil && p.str == "" {
		err = nil
	}

	return func() (float64, error) {
		return val, err
	}, err
}

var _ BoolGetter = (*constPlugin)(nil)

func (p *constPlugin) BoolGetter() (func() (bool, error), error) {
	val, err := strconv.ParseBool(p.str)
	if err != nil && p.str == "" {
		err = nil
	}

	return func() (bool, error) {
		return val, err
	}, err
}

var _ IntSetter = (*constPlugin)(nil)

func (p *constPlugin) IntSetter(param string) (func(int64) error, error) {
	set, err := p.set.IntSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}

	val, err := strconv.ParseInt(p.str, 10, 64)
	if err != nil && p.str == "" {
		err = nil
	}

	return func(_ int64) error {
		return set(val)
	}, err
}

var _ FloatSetter = (*constPlugin)(nil)

func (p *constPlugin) FloatSetter(param string) (func(float64) error, error) {
	set, err := p.set.FloatSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}

	val, err := strconv.ParseFloat(p.str, 64)
	if err != nil && p.str == "" {
		err = nil
	}

	return func(_ float64) error {
		return set(val)
	}, err
}

var _ BoolSetter = (*constPlugin)(nil)

func (p *constPlugin) BoolSetter(param string) (func(bool) error, error) {
	set, err := p.set.BoolSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}

	val, err := strconv.ParseBool(p.str)
	if err != nil && p.str == "" {
		err = nil
	}

	return func(_ bool) error {
		return set(val)
	}, err
}

var _ BytesSetter = (*constPlugin)(nil)

func (p *constPlugin) BytesSetter(param string) (func([]byte) error, error) {
	set, err := p.set.BytesSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}

	str := strings.ReplaceAll(strings.TrimPrefix(p.str, "0x"), "_", "")

	val, err := hex.DecodeString(str)
	if err != nil {
		err = nil
	}

	return func(_ []byte) error {
		return set(val)
	}, err
}
