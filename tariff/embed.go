package tariff

import "github.com/evcc-io/evcc/tariff/fixed"

type embed struct {
	Charges float64          `mapstructure:"charges"`
	Zones   fixed.ZoneConfig `mapstructure:"zones"`
	Tax     float64          `mapstructure:"tax"`
}

func (t *embed) totalPrice(price float64) float64 {
	return (price + t.Charges) * (1 + t.Tax)
}
