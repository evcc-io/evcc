package meter

import (
	"strings"

	"github.com/volkszaehler/mbmd/meters/rs485"
)

// isRS485 determines if model is a known MBMD rs485 device model
func isRS485(model string) bool {
	for k := range rs485.Producers {
		if strings.EqualFold(model, k) {
			return true
		}
	}
	return false
}
