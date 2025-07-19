package tariff

import (
	"github.com/evcc-io/evcc/api"
)

type cached struct {
	typ   api.TariffType
	rates api.Rates
}
