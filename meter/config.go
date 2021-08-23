package meter

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

var registry = make(util.Registry[api.Meter])

// NewFromConfig creates meter from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.Meter, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create meter '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid meter type: %s", typ)
	}

	return
}
