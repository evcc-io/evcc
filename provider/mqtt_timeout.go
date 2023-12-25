package provider

// TimeoutHandler is a wrapper for a Getter that times out after a given duration
type TimeoutHandler struct {
	ticker func() (string, error)
}

func NewTimeoutHandler(ticker func() (string, error)) *TimeoutHandler {
	return &TimeoutHandler{ticker}
}

func (h *TimeoutHandler) BoolGetter(p BoolProvider) (func() (bool, error), error) {
	g, err := p.BoolGetter()
	if err != nil {
		return nil, err
	}

	return func() (val bool, err error) {
		if val, err = g(); err == nil {
			_, err = h.ticker()
		}
		return val, err
	}, nil
}

func (h *TimeoutHandler) FloatGetter(p FloatProvider) (func() (float64, error), error) {
	g, err := p.FloatGetter()
	if err != nil {
		return nil, err
	}

	return func() (val float64, err error) {
		if val, err = g(); err == nil {
			_, err = h.ticker()
		}
		return val, err
	}, nil
}

func (h *TimeoutHandler) StringGetter(p StringProvider) (func() (string, error), error) {
	g, err := p.StringGetter()
	if err != nil {
		return nil, err
	}

	return func() (val string, err error) {
		if val, err = g(); err == nil {
			_, err = h.ticker()
		}
		return val, err
	}, nil
}
