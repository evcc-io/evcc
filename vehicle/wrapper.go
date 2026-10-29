package vehicle

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	corevehicle "github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/util"
)

// retryInterval is the minimum time between attempts to create an unavailable vehicle
const retryInterval = 15 * time.Minute

// Wrapper wraps an api.Vehicle that could not be created and retries creating it
type Wrapper struct {
	embed
	mu     sync.Mutex
	log    *util.Logger
	ctx    context.Context
	typ    string
	other  map[string]any
	config map[string]any

	vehicle   api.Vehicle
	err       error
	retriedAt time.Time // last creation attempt
}

var _ api.Delegator = (*Wrapper)(nil)

// NewWrapper creates a wrapper for a vehicle that could not be created
func NewWrapper(ctx context.Context, name, typ string, other map[string]any, err error) api.Vehicle {
	var cc struct {
		embed `mapstructure:",squash"`
		Other map[string]any `mapstructure:",remain"`
	}

	// try to decode vehicle-specific config and look for title attribute
	_ = util.DecodeOther(other, &cc)

	if cc.Title_ == "" {
		//lint:ignore SA1019 as Title is safe on ascii
		cc.Title_ = strings.Title(name)
	}

	v := &Wrapper{
		embed:     cc.embed,
		log:       util.ContextLoggerWithDefault(ctx, util.NewLogger("vehicle")),
		ctx:       ctx,
		typ:       typ,
		other:     other,
		config:    cc.Other,
		err:       fmt.Errorf("vehicle not available: %w", err),
		retriedAt: time.Now(), // creation has just failed
	}

	v.Features_ = append(v.Features_, api.Offline, api.Retryable)
	v.SetTitle(cc.Title_)

	return v
}

// WrappedConfig indicates a device with wrapped configuration
func (v *Wrapper) WrappedConfig() (string, map[string]any) {
	return v.typ, v.config
}

// instance returns the vehicle once created. Creation is retried at most once per retryInterval.
// The attempt runs in background since vehicle apis may block the control loop for minutes.
func (v *Wrapper) instance() api.Vehicle {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.vehicle == nil && time.Since(v.retriedAt) >= retryInterval {
		v.retriedAt = time.Now()
		go v.create()
	}

	return v.vehicle
}

func (v *Wrapper) create() {
	vehicle, err := NewFromConfig(v.ctx, v.typ, v.other)

	v.mu.Lock()
	if err != nil {
		v.err = fmt.Errorf("vehicle not available: %w", err)
		v.log.WARN.Printf("creating vehicle failed: %v", err)
	} else {
		if vehicle.GetTitle() == "" {
			vehicle.SetTitle(v.Title_)
		}
		v.vehicle = vehicle
		v.log.INFO.Printf("vehicle available: %s", util.TypeWithTemplateName(v.typ, v.other))
	}
	v.mu.Unlock()

	// publish updated features
	if err == nil && corevehicle.Publish != nil {
		corevehicle.Publish()
	}
}

// Delegate implements the api.Delegator interface
func (v *Wrapper) Delegate() any {
	if vv := v.instance(); vv != nil {
		return vv
	}
	return nil
}

// GetTitle implements the api.TitleDescriber interface
func (v *Wrapper) GetTitle() string {
	if vv := v.instance(); vv != nil {
		return vv.GetTitle()
	}
	return v.embed.GetTitle()
}

// SetTitle implements the api.TitleSetter interface
func (v *Wrapper) SetTitle(title string) {
	if vv := v.instance(); vv != nil {
		vv.SetTitle(title)
	}
	v.Title_ = title
}

// Features implements the api.FeatureDescriber interface
func (v *Wrapper) Features() []api.Feature {
	if vv := v.instance(); vv != nil {
		return vv.Features()
	}
	return v.embed.Features()
}

// Icon implements the api.IconDescriber interface
func (v *Wrapper) Icon() string {
	if vv := v.instance(); vv != nil {
		return vv.Icon()
	}
	return v.embed.Icon()
}

// Capacity implements the api.Vehicle interface
func (v *Wrapper) Capacity() float64 {
	if vv := v.instance(); vv != nil {
		return vv.Capacity()
	}
	return v.embed.Capacity()
}

// Phases implements the api.PhaseDescriber interface
func (v *Wrapper) Phases() int {
	if vv := v.instance(); vv != nil {
		return vv.Phases()
	}
	return v.embed.Phases()
}

// Identifiers implements the api.Vehicle interface
func (v *Wrapper) Identifiers() []string {
	if vv := v.instance(); vv != nil {
		return vv.Identifiers()
	}
	return v.embed.Identifiers()
}

// OnIdentified implements the api.Vehicle interface
func (v *Wrapper) OnIdentified() api.ActionConfig {
	if vv := v.instance(); vv != nil {
		return vv.OnIdentified()
	}
	return v.embed.OnIdentified()
}

var _ api.Battery = (*Wrapper)(nil)

// Soc implements the api.Battery interface
func (v *Wrapper) Soc() (float64, error) {
	if vv := v.instance(); vv != nil {
		return vv.Soc()
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	return 0, v.err
}
