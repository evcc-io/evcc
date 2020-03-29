package soc

import (
	"github.com/andig/evcc/api"
)

// NewFromConfig creates charger from configuration
func NewFromConfig(log *api.Logger, typ string, title string, other map[string]interface{}) api.SoC {
	var c api.SoC

	switch typ {
	case "script":
		c = NewConfigurableFromConfig(log, title, other)
	case "tesla":
		c = NewTeslaFromConfig(log, title, other)
	default:
		log.FATAL.Fatalf("invalid soc type '%s'", typ)
	}

	return c
}
