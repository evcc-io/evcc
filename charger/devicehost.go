package charger

import (
	"github.com/evcc-io/evcc/devicehost"
)

func init() {
	registry.AddCtx(devicehost.Type, devicehost.Charger)
}
