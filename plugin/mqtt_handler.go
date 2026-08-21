package plugin

import (
	"context"

	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
)

type msgHandler struct {
	ctx      context.Context
	topic    string
	pipeline *pipeline.Pipeline
	val      *util.Monitor[string]
}

func (h *msgHandler) receive(payload string) {
	h.val.Set(payload)
}

// value returns the received and processed payload as string
func (h *msgHandler) value() (string, error) {
	payload, err := h.val.GetContext(h.ctx)
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
