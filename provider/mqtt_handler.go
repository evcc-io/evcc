package provider

import (
	"fmt"
	"math"
	"strconv"

	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
)

type msgHandler struct {
	scale    float64
	topic    string
	pipeline *pipeline.Pipeline
	val      *util.Monitor[string]
}

func (h *msgHandler) receive(payload string) {
	h.val.Set(payload)
}

// hasValue returned the received and processed payload as string
func (h *msgHandler) hasValue() (string, error) {
	payload, err := h.val.Get()
	if err != nil {
		return "", err
	}

	if h.pipeline != nil {
		b, err := h.pipeline.Process([]byte(payload))
		return string(b), err
	}

	return payload, nil
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

	return v, nil
}

func (h *msgHandler) boolGetter() (bool, error) {
	v, err := h.hasValue()
	if err != nil {
		return false, err
	}

	return util.Truish(v), nil
}
