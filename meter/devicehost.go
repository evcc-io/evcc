package meter

import (
	"github.com/evcc-io/evcc/devicehost"
)

func init() {
	registry.AddCtx(devicehost.Type, devicehost.Meter)
}
