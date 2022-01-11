package tariff

import (
	"errors"
	"strings"

	"github.com/evcc-io/evcc/api"
)

// NewFromConfig creates new HEMS from config
func NewFromConfig(typ string, other map[string]interface{}) (t api.Tariff, err error) {
	switch strings.ToLower(typ) {
	case "fixed":
		t, err = NewFixed(other)
	case "awattar":
		t, err = NewAwattar(other)
	case "tibber":
		t, err = NewTibber(other)
	default:
		return nil, errors.New("unknown tariff: " + typ)
	}

	return
}
