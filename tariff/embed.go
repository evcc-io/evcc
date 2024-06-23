package tariff

type embed struct {
	Charges float64 `mapstructure:"charges"`
	Tax     float64 `mapstructure:"tax"`
}

func (t *embed) totalPrice(price float64) float64 {
	total := price + t.Charges
	if total > 0 {
		total *= 1 + t.Tax
	}
	return total
}
