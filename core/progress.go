package core

type Progress struct {
	min, step float64
}

func NewProgress(min, step float64) *Progress {
	return &Progress{
		min:  min,
		step: step,
	}
}

func (p *Progress) NextStep(value float64) bool {
	if value >= p.min {
		for p.min <= value {
			p.min += p.step
		}

		return true
	}

	return false
}
