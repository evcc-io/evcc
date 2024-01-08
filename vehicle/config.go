package vehicle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	expiry   = 5 * time.Minute  // maximum response age before refresh
	interval = 15 * time.Minute // refresh interval when charging
)

type (
	factoryFunc     func(context.Context, map[string]interface{}) (api.Vehicle, error)
	vehicleRegistry map[string]factoryFunc
)

func (r vehicleRegistry) Add(name string, factory func(map[string]interface{}) (api.Vehicle, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate vehicle type: %s", name))
	}
	r[name] = func(_ context.Context, cc map[string]interface{}) (api.Vehicle, error) {
		return factory(cc)
	}
}

func (r vehicleRegistry) Get(name string) (factoryFunc, error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("invalid vehicle type: %s", name)
	}
	return factory, nil
}

var registry vehicleRegistry = make(map[string]factoryFunc)

// Types returns the list of vehicle types
func Types() []string {
	var res []string
	for typ := range registry {
		res = append(res, typ)
	}
	return res
}

// NewFromConfig creates vehicle from configuration
func NewFromConfig(ctx context.Context, typ string, other map[string]interface{}) (api.Vehicle, error) {
	var cc struct {
		Cloud bool
		Other map[string]interface{} `mapstructure:",remain"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Cloud {
		cc.Other["brand"] = typ
		typ = "cloud"
	}

	factory, err := registry.Get(strings.ToLower(typ))
	if err != nil {
		return nil, err
	}

	v, err := factory(ctx, cc.Other)
	if err != nil {
		err = fmt.Errorf("cannot create vehicle type '%s': %w", typ, err)
	}

	return v, err
}
