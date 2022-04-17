package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	persist "github.com/evcc-io/evcc/util/store"
)

const (
	expiry   = 5 * time.Minute  // maximum response age before refresh
	interval = 15 * time.Minute // refresh interval when charging
)

type (
	factoryFunc   = func(map[string]interface{}) (api.Vehicle, error)
	factoryFuncEx = func(map[string]interface{}, persist.Store) (api.Vehicle, error)

	vehicleRegistry map[string]factoryFuncEx
)

func (r vehicleRegistry) Add(name string, factory factoryFunc) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate vehicle type: %s", name))
	}
	r[name] = func(cc map[string]interface{}, _ persist.Store) (api.Vehicle, error) {
		return factory(cc)
	}
}

func (r vehicleRegistry) AddWithStore(name string, factory factoryFuncEx) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate vehicle type: %s", name))
	}
	r[name] = factory
}

func (r vehicleRegistry) Get(name string) (factoryFuncEx, error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("vehicle type not registered: %s", name)
	}
	return factory, nil
}

var (
	store                    = persist.New("vehicles")
	registry vehicleRegistry = make(map[string]factoryFuncEx)
)

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
	cc := struct {
		Cloud bool
		Other map[string]interface{} `mapstructure:",remain"`
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Cloud {
		cc.Other["brand"] = typ
		typ = "cloud"
	}

	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(cc.Other, store); err != nil {
			err = fmt.Errorf("cannot create vehicle '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid vehicle type: %s", typ)
	}

	return
}
