// +build !linux

package charger

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

// NewNRGKickBLEFromConfig creates a NRGKickBLE charger from generic config
func NewNRGKickBLEFromConfig(log *util.Logger, other map[string]interface{}) api.Charger {
	log.FATAL.Fatal("config: NRGKick bluetooth is only supported on linux")
	return nil
}
