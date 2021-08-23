package charger

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

var registry = make(util.Registry[api.Charger])

// NewFromConfig creates charger from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.Charger, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create charger '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid charger type: %s", typ)
	}

	return
}
