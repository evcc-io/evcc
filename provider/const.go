package provider

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
)

type constProvider struct {
	*getter
	ctx context.Context
	str string
	set Config
}

func init() {
	registry.AddCtx("const", NewConstFromConfig)
}

// NewConstFromConfig creates const provider
func NewConstFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	var cc struct {
		Value             string
		pipeline.Settings `mapstructure:",squash"`
		Set               Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	pipe, err := pipeline.New(nil, cc.Settings)
	if err != nil {
		return nil, err
	}

	b, err := pipe.Process([]byte(cc.Value))
	if err != nil {
		return nil, err
	}

	p := &constProvider{
		ctx: ctx,
		str: string(b),
		set: cc.Set,
	}

	p.getter = defaultGetters(p, 1)

	return p, nil
}

var _ StringProvider = (*constProvider)(nil)

func (p *constProvider) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		return p.str, nil
	}, nil
}

var _ SetIntProvider = (*constProvider)(nil)

func (p *constProvider) IntSetter(param string) (func(int64) error, error) {
	set, err := NewIntSetterFromConfig(p.ctx, param, p.set)
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

var _ SetFloatProvider = (*constProvider)(nil)

func (p *constProvider) FloatSetter(param string) (func(float64) error, error) {
	set, err := NewFloatSetterFromConfig(p.ctx, param, p.set)
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

var _ SetBoolProvider = (*constProvider)(nil)

func (p *constProvider) BoolSetter(param string) (func(bool) error, error) {
	set, err := NewBoolSetterFromConfig(p.ctx, param, p.set)
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

var _ SetBytesProvider = (*constProvider)(nil)

func (p *constProvider) BytesSetter(param string) (func([]byte) error, error) {
	set, err := NewBytesSetterFromConfig(p.ctx, param, p.set)
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
