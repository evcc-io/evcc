package vehicle

import (
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

// NewFromConfig creates vehicle from configuration
func NewFromConfig(log *util.Logger, typ string, other map[string]interface{}) api.Vehicle {
	var c api.Vehicle

	switch strings.ToLower(typ) {
	case "default", "configurable":
		c = NewConfigurableFromConfig(log, other)
	case "audi", "etron":
		c = NewAudiFromConfig(log, other)
	case "bmw", "i3":
		c = NewBMWFromConfig(log, other)
	case "tesla", "model3", "model 3", "models", "model s":
		c = NewTeslaFromConfig(log, other)
	case "nissan", "leaf":
		c = NewNissanFromConfig(log, other)
	default:
		log.FATAL.Fatalf("invalid vehicle type '%s'", typ)
	}

	return c
}
