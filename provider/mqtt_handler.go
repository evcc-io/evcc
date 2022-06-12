package provider

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
)

type msgHandler struct {
	mux      sync.Mutex
	wait     *util.Waiter
	scale    float64
	topic    string
	pipeline *pipeline.Pipeline
	payload  string
}

func (h *msgHandler) receive(payload string) {
	h.mux.Lock()
	defer h.mux.Unlock()

	h.payload = payload
	h.wait.Update()
}

// hasValue returned the received and processed payload as string
func (h *msgHandler) hasValue() (string, error) {
	if late := h.wait.Overdue(); late > 0 {
		return "", fmt.Errorf("%s outdated: %v", h.topic, late.Truncate(time.Second))
	}

	h.mux.Lock()
	defer h.mux.Unlock()

	if h.pipeline != nil {
		b, err := h.pipeline.Process([]byte(h.payload))
		return string(b), err
	}

	return h.payload, nil
}

func (h *msgHandler) floatGetter() (float64, error) {
	v, err := h.hasValue()
	if err != nil {
		return 0, err
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s invalid: '%s'", h.topic, v)
	}

	return f * h.scale, nil
}

func (h *msgHandler) intGetter() (int64, error) {
	f, err := h.floatGetter()
	return int64(math.Round(f)), err
}

func (h *msgHandler) stringGetter() (string, error) {
	v, err := h.hasValue()
	if err != nil {
		return "", err
	}

	return string(v), nil
}

func (h *msgHandler) boolGetter() (bool, error) {
	v, err := h.hasValue()
	if err != nil {
		return false, err
	}

	return util.Truish(v), nil
}
