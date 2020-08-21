package charger

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
)

type apiFunction string

// NewFromConfig creates charger from configuration
func NewFromConfig(typ string, other map[string]interface{}) (charger api.Charger, err error) {
	switch strings.ToLower(typ) {
	case "default", "configurable":
		charger, err = NewConfigurableFromConfig(other)
	case "wallbe":
		charger, err = NewWallbeFromConfig(other)
	case "phoenix-emcp":
		charger, err = NewPhoenixEMCPFromConfig(other)
	case "phoenix-evcc":
		charger, err = NewPhoenixEVCCFromConfig(other)
	case "nrgkick-bluetooth", "nrgkick-bt", "nrgble":
		charger, err = NewNRGKickBLEFromConfig(other)
	case "nrgkick-connect", "nrgconnect":
		charger, err = NewNRGKickConnectFromConfig(other)
	case "go-e", "goe":
		charger, err = NewGoEFromConfig(other)
	case "evsewifi":
		charger, err = NewEVSEWifiFromConfig(other)
	case "simpleevse", "evse":
		charger, err = NewSimpleEVSEFromConfig(other)
	case "porsche", "audi", "bentley", "mcc":
		charger, err = NewMobileConnectFromConfig(other)
	case "keba", "bmw":
		charger, err = NewKebaFromConfig(other)
	default:
		return nil, fmt.Errorf("invalid charger type: %s", typ)
	}

	if err != nil {
		err = fmt.Errorf("cannot create %s charger: %v", typ, err)
	}

	return charger, err
}
