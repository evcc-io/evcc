package core

type circuitLimiter struct {
	maxCurrent float64
	loadpoints map[interface{}]float64
}

var circuit *circuitLimiter

func init() {
	circuit = &circuitLimiter{
		maxCurrent: 16,
		loadpoints: make(map[interface{}]float64),
	}
}

func (lim *circuitLimiter) Limit(owner interface{}, current float64, enabled bool) float64 {
	var sum float64
	for lp, current := range lim.loadpoints {
		if lp != owner {
			sum += current
		}
	}

	if sum+current > lim.maxCurrent {
		current = lim.maxCurrent - sum
	}

	return current
}

func (lim *circuitLimiter) Update(owner interface{}, current float64, enabled bool) {
	if !enabled {
		current = 0
	}

	lim.loadpoints[owner] = current
}
