package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dario.cat/mergo"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Collect collects and combines JSON maps
type Collect struct {
	*getter
	log     *util.Logger
	get     func() (string, error)
	data    map[string]any
	cache   time.Duration
	timeout time.Duration
	updated time.Time
}

func init() {
	registry.AddCtx("collect", NewCollectProviderFromConfig)
}

// NewCollectProviderFromConfig creates a collect provider.
func NewCollectProviderFromConfig(ctx context.Context, other map[string]interface{}) (Provider, error) {
	cc := struct {
		Get     Config
		Timeout time.Duration
		Cache   time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	g, err := NewStringGetterFromConfig(ctx, cc.Get)
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	p, err := NewCollectProvider(g, cc.Timeout, cc.Cache)
	p.getter = defaultGetters(p, 1)

	return p, err
}

// NewCollectProvider creates a collect provider.
// Collect execution is aborted after given timeout.
func NewCollectProvider(get func() (string, error), timeout time.Duration, cache time.Duration) (*Collect, error) {
	s := &Collect{
		log:     util.NewLogger("collect"),
		get:     get,
		data:    make(map[string]any),
		cache:   cache,
		timeout: timeout,
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
