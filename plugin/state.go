package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
)

// State reads a published value from the process-wide value cache (the same store
// that backs /api/state) and forwards it to a nested setter. A jq filter navigates
// nested values. This lets templates write a site-computed value to a device
// register in-process, without an HTTP round-trip to the local API.
type State struct {
	ctx       context.Context
	key       string
	scale     float64
	pipeline  *pipeline.Pipeline
	setConfig *Config
}

func init() {
	registry.AddCtx("state", NewStateFromConfig)
}

// NewStateFromConfig creates a state provider
func NewStateFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Key               string
		pipeline.Settings `mapstructure:",squash"`
		Scale             float64
		Set               *Config
	}{
		Scale: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Key == "" {
		return nil, errors.New("missing key")
	}

	pipe, err := pipeline.New(util.ContextLoggerWithDefault(ctx, util.NewLogger("state")), cc.Settings)
	if err != nil {
		return nil, err
	}

	return &State{
		ctx:       ctx,
		key:       cc.Key,
		scale:     cc.Scale,
		pipeline:  pipe,
		setConfig: cc.Set,
	}, nil
}

// value reads the cached value, applies the jq pipeline and returns it as float64.
// Returns 0 when the key has not been published yet (the value source is absent).
//
// A null or empty jq result is surfaced as an error rather than silently mapped to
// a value: defaulting is consumer policy, not transport behavior, and swallowing it
// would hide a mistyped key/filter. Consumers that want a default must express it in
// the jq, e.g. `(.foo) // 0`.
func (p *State) value() (float64, error) {
	v := util.DefaultParamCacheValue(p.key)
	if v == nil {
		return 0, nil
	}

	b, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}

	if b, err = p.pipeline.Process(b); err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
	if err != nil {
		return 0, err
	}
	return f * p.scale, nil
}

var _ FloatGetter = (*State)(nil)

func (p *State) FloatGetter() (func() (float64, error), error) {
	return p.value, nil
}

func (p *State) forward(param string) (func() error, error) {
	if p.setConfig == nil {
		return nil, errors.New("missing set config")
	}
	set, err := p.setConfig.FloatSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}
	return func() error {
		v, err := p.value()
		if err != nil {
			return err
		}
		return set(v)
	}, nil
}

var _ IntSetter = (*State)(nil)

// IntSetter ignores the input and forwards the cached value to the nested setter.
func (p *State) IntSetter(param string) (func(int64) error, error) {
	fwd, err := p.forward(param)
	if err != nil {
		return nil, err
	}
	return func(int64) error { return fwd() }, nil
}

var _ FloatSetter = (*State)(nil)

// FloatSetter ignores the input and forwards the cached value to the nested setter.
func (p *State) FloatSetter(param string) (func(float64) error, error) {
	fwd, err := p.forward(param)
	if err != nil {
		return nil, err
	}
	return func(float64) error { return fwd() }, nil
}
