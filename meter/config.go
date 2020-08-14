package meter

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/pkg/errors"
)

// NewFromConfig creates meter from configuration
func NewFromConfig(typ string, other map[string]interface{}) (meter api.Meter, err error) {
	switch strings.ToLower(typ) {
	case "default", "configurable":
		meter, err = NewConfigurableFromConfig(other)
	case "modbus":
		meter, err = NewModbusFromConfig(other)
	case "sma":
		meter, err = NewSMAFromConfig(other)
	case "tesla", "powerwall":
		meter, err = NewTeslaFromConfig(other)
	default:
		err = fmt.Errorf("invalid meter type: %s", typ)
	}

	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("cannot create %s meter", typ))
	}

	return meter, err
}
