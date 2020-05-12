package charger

import (
	"strings"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

type apiFunction string

// NewFromConfig creates charger from configuration
func NewFromConfig(log *util.Logger, typ string, other map[string]interface{}) api.Charger {
	var c api.Charger

	switch strings.ToLower(typ) {
	case "default", "configurable":
		c = NewConfigurableFromConfig(log, other)
	case "wallbe":
		c = NewWallbeFromConfig(log, other)
	case "phoenix-emcp":
		c = NewPhoenixEMCPFromConfig(log, other)
	case "phoenix-evcc":
		c = NewPhoenixEVCCFromConfig(log, other)
	case "nrgkick-bluetooth", "nrgkick-bt", "nrgble":
		c = NewNRGKickBLEFromConfig(log, other)
	case "nrgkick-connect", "nrgconnect":
		c = NewNRGKickConnectFromConfig(log, other)
	case "go-e", "goe":
		c = NewGoEFromConfig(log, other)
	case "evsewifi":
		c = NewEVSEWifiFromConfig(log, other)
	case "simpleevse", "evse":
		c = NewSimpleEVSEFromConfig(log, other)
	case "porsche", "audi", "bentley", "mcc":
		c = NewMobileConnectFromConfig(log, other)
	case "keba", "bmw":
		c = NewKebaFromConfig(log, other)
	default:
		log.FATAL.Fatalf("invalid charger type '%s'", typ)
	}

	return c
}
