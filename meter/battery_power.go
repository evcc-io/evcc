package meter

type maxpower struct {
	MaxACPower float64
}

// var _ api.MaxACPowerGetter = (*maxpower)(nil)

// MaxACPowerDecorator returns the max AC power decorator
func (m *maxpower) Decorator() func() float64 {
	if m.MaxACPower == 0 {
		return nil
	}
	return func() float64 {
		return m.MaxACPower
	}
}
