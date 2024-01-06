package tariff

type embed struct {
	Charges float64 `mapstructure:"charges"`
	Tax     float64 `mapstructure:"tax"`
}

func (t *embed) totalPrice(price float64) float64 {
	return (price + t.Charges) * (1 + t.Tax)
}
