package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
)

const (
	expiry   = 5 * time.Minute  // maximum response age before refresh
	interval = 15 * time.Minute // refresh interval when charging
)

type (
	vehicleRegistry  map[string]factoryFunc
	factoryFunc      func(store.Provider, map[string]any) (api.Vehicle, error)
	factoryFuncPlain func(map[string]any) (api.Vehicle, error)
)

func (r vehicleRegistry) Add(name string, f factoryFuncPlain) {
	factory := func(_ store.Provider, other map[string]any) (api.Vehicle, error) {
		return f(other)
	}

	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate vehicle type: %s", name))
	}
	r[name] = factory
}

func (r vehicleRegistry) AddWithStore(name string, factory factoryFunc) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate vehicle type: %s", name))
	}
	r[name] = factory
}

func (r vehicleRegistry) Get(name string) (factoryFunc, error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("vehicle type not registered: %s", name)
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
func NewFromConfig(typ string, other map[string]interface{}) (v api.Vehicle, err error) {
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
	if err == nil {
		storeFactory := func(string) store.Store { return nil }
		if db.Instance != nil {
			storeFactory = settings.NewStore
		}

		if v, err = factory(storeFactory, cc.Other); err != nil {
			err = fmt.Errorf("cannot create vehicle '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid vehicle type: %s", typ)
	}

	return
}
