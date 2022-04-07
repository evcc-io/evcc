package provider

// TimeoutHandler is a wrapper for a Getter that times out after a given duration
type TimeoutHandler struct {
	ticker func() (string, error)
}

func NewTimeoutHandler(ticker func() (string, error)) *TimeoutHandler {
	return &TimeoutHandler{ticker}
}

func (h *TimeoutHandler) BoolGetter(g func() (bool, error)) func() (bool, error) {
	return func() (val bool, err error) {
		if val, err = g(); err == nil {
			_, err = h.ticker()
		}
		return val, err
	}
}

func (h *TimeoutHandler) FloatGetter(g func() (float64, error)) func() (float64, error) {
	return func() (val float64, err error) {
		if val, err = g(); err == nil {
			_, err = h.ticker()
		}
		return val, err
	}
}

func (h *TimeoutHandler) StringGetter(g func() (string, error)) func() (string, error) {
	return func() (val string, err error) {
		if val, err = g(); err == nil {
			_, err = h.ticker()
		}
		return val, err
	}
}
