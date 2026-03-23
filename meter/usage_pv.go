package meter

type pvMaxACPower struct {
	MaxACPower float64
}

// var _ api.MaxACPowerGetter = (*pvMaxACPower)(nil)

// Decorator returns the max AC power decorator
func (m *pvMaxACPower) Decorator() func() float64 {
	if m.MaxACPower == 0 {
		return nil
	}
	return func() float64 {
		return m.MaxACPower
	}
}
