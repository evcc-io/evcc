package vehicle

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/wrapper"
)

const interval = 15 * time.Minute

var registry = make(util.Registry[api.Vehicle])

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
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			// wrap any created errors to prevent fatals
			v, err = wrapper.New(v, err)
		}
	} else {
		err = fmt.Errorf("invalid vehicle type: %s", typ)
	}

	return
}
