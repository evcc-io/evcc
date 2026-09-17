package plugin

import (
	"context"
	"errors"
	"sync"

	"github.com/evcc-io/evcc/util"
)

// Memory holds a named value in a device-local store. Without a nested setter it sinks
// the incoming value; with one it forwards the stored value instead, e.g. into a batterymode
// switch case.
type Memory struct {
	ctx       context.Context
	store     *memoryStore
	name      string
	setConfig *Config
}

// memoryStore is a device-local, name-addressed value store shared by all Memory
// plugins built from the same context.
type memoryStore struct {
	mu sync.RWMutex
	m  map[string]float64
}

func (s *memoryStore) set(name string, v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[name] = v
}

func (s *memoryStore) get(name string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[name]
}

type memoryStoreKey struct{}

// WithMemoryStore returns a context carrying a fresh device-local memory store, so Memory
// plugins built from it share a store while other devices stay isolated.
func WithMemoryStore(ctx context.Context) context.Context {
	return context.WithValue(ctx, memoryStoreKey{}, &memoryStore{m: make(map[string]float64)})
}

// memoryStoreFromContext returns the device-local store, or an unshared standalone one
// if ctx was not scoped by WithMemoryStore.
func memoryStoreFromContext(ctx context.Context) *memoryStore {
	if s, ok := ctx.Value(memoryStoreKey{}).(*memoryStore); ok {
		return s
	}
	return &memoryStore{m: make(map[string]float64)}
}

func init() {
	registry.AddCtx("memory", NewMemoryFromConfig)
}

// NewMemoryFromConfig creates a memory provider
func NewMemoryFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	cc := struct {
		Name    string
		Set     *Config
		Initial *float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Name == "" {
		return nil, errors.New("missing name")
	}

	store := memoryStoreFromContext(ctx)

	// seed the cell so a control path reading it before the first push gets a sane
	// value instead of zero
	if cc.Initial != nil {
		store.set(cc.Name, *cc.Initial)
	}

	return &Memory{
		ctx:       ctx,
		store:     store,
		name:      cc.Name,
		setConfig: cc.Set,
	}, nil
}

var _ FloatGetter = (*Memory)(nil)

func (p *Memory) FloatGetter() (func() (float64, error), error) {
	return func() (float64, error) {
		return p.store.get(p.name), nil
	}, nil
}

func (p *Memory) forward(param string) (func() error, error) {
	set, err := p.setConfig.FloatSetter(p.ctx, param)
	if err != nil {
		return nil, err
	}
	return func() error {
		return set(p.store.get(p.name))
	}, nil
}

var _ FloatSetter = (*Memory)(nil)

// FloatSetter stores the incoming value (sink), or forwards the stored value to the
// nested setter ignoring the input (forward), depending on whether set is configured.
func (p *Memory) FloatSetter(param string) (func(float64) error, error) {
	if p.setConfig == nil {
		return func(val float64) error {
			p.store.set(p.name, val)
			return nil
		}, nil
	}

	fwd, err := p.forward(param)
	if err != nil {
		return nil, err
	}
	return func(float64) error { return fwd() }, nil
}

var _ IntSetter = (*Memory)(nil)

// IntSetter mirrors FloatSetter for int-typed control paths (e.g. a batterymode switch).
func (p *Memory) IntSetter(param string) (func(int64) error, error) {
	if p.setConfig == nil {
		return func(val int64) error {
			p.store.set(p.name, float64(val))
			return nil
		}, nil
	}

	fwd, err := p.forward(param)
	if err != nil {
		return nil, err
	}
	return func(int64) error { return fwd() }, nil
}
