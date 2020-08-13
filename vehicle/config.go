package vehicle

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
	"github.com/pkg/errors"
)

// NewFromConfig creates vehicle from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.Vehicle, err error) {
	switch strings.ToLower(typ) {
	case "default", "configurable":
		v, err = NewConfigurableFromConfig(other)
	case "audi", "etron":
		v, err = NewAudiFromConfig(other)
	case "bmw", "i3":
		v, err = NewBMWFromConfig(other)
	case "tesla", "model3", "model 3", "models", "model s":
		v, err = NewTeslaFromConfig(other)
	case "nissan", "leaf":
		v, err = NewNissanFromConfig(other)
	case "renault", "zoe":
		v, err = NewRenaultFromConfig(other)
	case "porsche", "taycan":
		v, err = NewPorscheFromConfig(other)
	default:
		err = fmt.Errorf("invalid vehicle type: %s", typ)
	}

	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("cannot create %s vehicle", typ))
	}

	return
}
