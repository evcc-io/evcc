package meter

import (
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

// NewFromConfig creates meter from configuration
func NewFromConfig(log *util.Logger, typ string, other map[string]interface{}) api.Meter {
	var c api.Meter

	switch strings.ToLower(typ) {
	case "default", "configurable":
		c = NewConfigurableFromConfig(log, other)
	case "modbus":
		c = NewModbusFromConfig(log, other)
	case "sma":
		c = NewSMAFromConfig(log, other)
	case "tesla", "powerwall":
		c = NewTeslaFromConfig(log, other)
	default:
		log.FATAL.Fatalf("invalid meter type '%s'", typ)
	}

	return c
}
