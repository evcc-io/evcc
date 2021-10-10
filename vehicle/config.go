package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
)

const (
	expiry   = 5 * time.Minute  // maximum response age before refresh
	interval = 15 * time.Minute // refresh interval when charging
)

type vehicleRegistry map[string]*meta

type meta struct {
	factory func(map[string]interface{}) (api.Vehicle, error)
	Config  interface{}
}

func (r vehicleRegistry) Add(name string, factory func(map[string]interface{}) (api.Vehicle, error), Config interface{}) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate vehicle type: %s", name))
	}
	r[name] = &meta{factory, Config}
}

func (r vehicleRegistry) Get(name string) (*meta, error) {
	meta, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("vehicle type not registered: %s", name)
	}
	return meta, nil
}

var registry vehicleRegistry = make(map[string]*meta)
var Registry = registry

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
	meta, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = meta.factory(other); err != nil {
			err = fmt.Errorf("cannot create vehicle '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid vehicle type: %s", typ)
	}

	return
}
