package meter

type capacity struct {
	Capacity float64
}

// var _ api.BatteryCapacity = (*capacity)(nil)

// Decorator returns the capacity decorator if capacity is specified or nil otherwise
func (m *capacity) Decorator() func() float64 {
	if m.Capacity == 0 {
		return nil
	}
	return func() float64 {
		return m.Capacity
	}
}
