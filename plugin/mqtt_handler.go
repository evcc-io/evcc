package plugin

import (
	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
)

type msgHandler struct {
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

	if err := knownErrors([]byte(payload)); err != nil {
		return "", err
	}

	if h.pipeline != nil {
		b, err := h.pipeline.Process([]byte(payload))
		return string(b), err
	}

	return payload, nil
}

func (h *msgHandler) value() (string, error) {
	v, err := h.hasValue()
	if err != nil {
		return "", err
	}

	return v, nil
}
