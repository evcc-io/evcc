package meter

import (
	"strings"

	"github.com/andig/evcc/api"
)

// NewFromConfig creates meter from configuration
func NewFromConfig(log *api.Logger, typ string, other map[string]interface{}) api.Meter {
	var c api.Meter

	switch strings.ToLower(typ) {
	case "default", "configurable":
		c = NewConfigurableFromConfig(log, other)
	case "smameter":
		c = NewSMAFromConfig(log, other)
	default:
		log.FATAL.Fatalf("invalid meter type '%s'", typ)
	}

	return c
}
