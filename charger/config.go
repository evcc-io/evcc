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
	case "phoenix":
		c = NewPhoenixFromConfig(log, other)
	case "nrgkick", "nrg", "kick":
		c = NewNRGKickFromConfig(log, other)
	case "go-e", "goe":
		c = NewGoEFromConfig(log, other)
	case "evsewifi":
		c = NewEVSEWifiFromConfig(log, other)
	case "simpleevse", "evse":
		c = NewSimpleEVSEFromConfig(log, other)
	case "porsche", "audi", "bentley", "mcc":
		c = NewMobileConnectFromConfig(log, other)
	default:
		log.FATAL.Fatalf("invalid charger type '%s'", typ)
	}

	return c
}
