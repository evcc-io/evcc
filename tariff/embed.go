package tariff

import "github.com/evcc-io/evcc/tariff/fixed"

type embed struct {
	Charges float64          `mapstructure:"charges"`
	Zones   fixed.ZoneConfig `mapstructure:"zones"`
	Tax     float64          `mapstructure:"tax"`
	zones   fixed.Zones
}

func (t *embed) parse() error {
	zz, err := t.Zones.Parse(t.Charges)
	if err == nil {
		t.zones = zz
	}
	return err
}

func (t *embed) totalPrice(price float64) float64 {
	return (price + t.Charges) * (1 + t.Tax)
}

// TODO remove
func (t *embed) totalPriceZonesCharges(price, charges float64) float64 {
	return (price + charges) * (1 + t.Tax)
}
