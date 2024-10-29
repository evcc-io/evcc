package tariff

type embed struct {
	Margin  float64 `mapstructure:"margin"`
	Charges float64 `mapstructure:"charges"`
	Tax     float64 `mapstructure:"tax"`
	Uplifts float64 `mapstructure:"uplifts"`
}

func (t *embed) totalPrice(price float64) float64 {
	return (price*(1+t.Margin)+t.Charges)*(1+t.Tax) + t.Uplifts
}
