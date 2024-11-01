package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/spf13/cast"
)

// Collect collects and combines JSON maps
type Collect struct {
	log     *util.Logger
	get     func() (string, error)
	data    map[string]any
	cache   time.Duration
	timeout time.Duration
	updated time.Time
	scale   float64
}

func init() {
	registry.AddCtx("collect", NewCollectProviderFromConfig)
}

// NewCollectProviderFromConfig creates a collect provider.
func NewCollectProviderFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	cc := struct {
		Get     Config
		Scale   float64
		Timeout time.Duration
		Cache   time.Duration
	}{
		Timeout: request.Timeout,
		Scale:   1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	g, err := NewStringGetterFromConfig(ctx, cc.Get)
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	p, err := NewCollectProvider(g, cc.Timeout, cc.Scale, cc.Cache)

	return p, err
}

// NewCollectProvider creates a collect provider.
// Collect execution is aborted after given timeout.
func NewCollectProvider(get func() (string, error), timeout time.Duration, scale float64, cache time.Duration) (*Collect, error) {
	s := &Collect{
		log:     util.NewLogger("collect"),
		get:     get,
		data:    make(map[string]any),
		cache:   cache,
		timeout: timeout,
		scale:   scale,
	}

	return s, nil
}

func (p *Collect) update() error {
	v, err := p.get()
	if err != nil {
		return err
	}

	var new map[string]any
	if err := json.Unmarshal([]byte(v), &new); err != nil {
		return err
	}

	return mergo.Merge(&p.data, v, mergo.WithOverride)
}

var _ StringProvider = (*Collect)(nil)

// StringGetter returns string from exec result. Only STDOUT is considered.
func (p *Collect) StringGetter() (func() (string, error), error) {
	return func() (string, error) {
		if err := p.update(); err != nil {
			return "", err
		}

		b, err := json.Marshal(p.data)
		return string(b), err
	}, nil
}

var _ FloatProvider = (*Collect)(nil)

// FloatGetter parses float from exec result
func (p *Collect) FloatGetter() (func() (float64, error), error) {
	g, err := p.StringGetter()

	return func() (float64, error) {
		s, err := g()
		if err != nil {
			return 0, err
		}

		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			f *= p.scale
		}

		return f, err
	}, err
}

var _ IntProvider = (*Collect)(nil)

// IntGetter parses int64 from exec result
func (p *Collect) IntGetter() (func() (int64, error), error) {
	g, err := p.FloatGetter()

	return func() (int64, error) {
		f, err := g()
		return int64(math.Round(f)), err
	}, err
}

var _ BoolProvider = (*Collect)(nil)

// BoolGetter parses bool from exec result. "on", "true" and 1 are considered truish.
func (p *Collect) BoolGetter() (func() (bool, error), error) {
	g, err := p.StringGetter()

	return func() (bool, error) {
		s, err := g()
		if err != nil {
			return false, err
		}

		return cast.ToBoolE(s)
	}, err
}
