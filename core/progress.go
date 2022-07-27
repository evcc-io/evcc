package core

type Progress struct {
	min, step, current float64
}

func NewProgress(min, step float64) *Progress {
	return &Progress{
		min:     min,
		step:    step,
		current: min,
	}
}

func (p *Progress) NextStep(value float64) bool {
	// test guard
	if p != nil && value >= p.current {
		for p.current <= value {
			p.current += p.step
		}

		return true
	}

	return false
}

func (p *Progress) Reset() {
	// test guard
	if p != nil {
		p.current = p.min
	}
}
