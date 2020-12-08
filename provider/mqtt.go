package provider

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/andig/evcc/util"
)

// Mqtt provider
type Mqtt struct {
	log    *util.Logger
	client *MqttClient
}

// NewMqttFromConfig creates Mqtt provider
func NewMqttFromConfig(other map[string]interface{}) (IntProvider, error) {
	cc := struct {
		Topic, Payload string // Payload only applies to setters
		Scale          float64
		Timeout        time.Duration
	}{
		Scale: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return nil, nil
}

// FloatGetter creates handler for float64 from MQTT topic that returns cached value
func (m *Mqtt) FloatGetter(topic string, scale float64, timeout time.Duration) func() (float64, error) {
	h := &msgHandler{
		log:   m.log,
		mux:   util.NewWaiter(timeout, func() { m.log.TRACE.Printf("%s wait for initial value", topic) }),
		topic: topic,
		scale: scale,
	}

	m.client.Listen(topic, h.Receive)
	return h.floatGetter
}

// IntGetter creates handler for int64 from MQTT topic that returns cached value
func (m *Mqtt) IntGetter(topic string, scale int64, timeout time.Duration) func() (int64, error) {
	h := &msgHandler{
		log:   m.log,
		mux:   util.NewWaiter(timeout, func() { m.log.TRACE.Printf("%s wait for initial value", topic) }),
		topic: topic,
		scale: float64(scale),
	}

	m.client.Listen(topic, h.Receive)
	return h.intGetter
}

// StringGetter creates handler for string from MQTT topic that returns cached value
func (m *Mqtt) StringGetter(topic string, timeout time.Duration) func() (string, error) {
	h := &msgHandler{
		log:   m.log,
		mux:   util.NewWaiter(timeout, func() { m.log.TRACE.Printf("%s wait for initial value", topic) }),
		topic: topic,
	}

	m.client.Listen(topic, h.Receive)
	return h.stringGetter
}

// BoolGetter creates handler for string from MQTT topic that returns cached value
func (m *Mqtt) BoolGetter(topic string, timeout time.Duration) func() (bool, error) {
	h := &msgHandler{
		log:   m.log,
		mux:   util.NewWaiter(timeout, func() { m.log.TRACE.Printf("%s wait for initial value", topic) }),
		topic: topic,
	}

	m.client.Listen(topic, h.Receive)
	return h.boolGetter
}

// formatValue formats a message template of returns the value formatted as %v is template is empty
func (m *Mqtt) formatValue(param, message string, v interface{}) (string, error) {
	if message == "" {
		return fmt.Sprintf("%v", v), nil
	}

	return util.ReplaceFormatted(message, map[string]interface{}{
		param: v,
	})
}

// IntSetter publishes topic with parameter replaced by int value
func (m *Mqtt) IntSetter(param, topic, message string) func(int64) error {
	return func(v int64) error {
		payload, err := m.formatValue(param, message, v)
		if err != nil {
			return err
		}

		m.log.TRACE.Printf("send %s: '%s'", topic, payload)
		return m.client.Publish(topic, false, payload)
	}
}

// BoolSetter invokes script with parameter replaced by bool value
func (m *Mqtt) BoolSetter(param, topic, message string) func(bool) error {
	return func(v bool) error {
		payload, err := m.formatValue(param, message, v)
		if err != nil {
			return err
		}

		m.log.TRACE.Printf("send %s: '%s'", topic, payload)
		return m.client.Publish(topic, false, payload)
	}
}

type msgHandler struct {
	log     *util.Logger
	mux     *util.Waiter
	scale   float64
	topic   string
	payload string
}

func (h *msgHandler) Receive(payload string) {
	h.log.TRACE.Printf("recv %s: '%s'", h.topic, payload)

	h.mux.Lock()
	defer h.mux.Unlock()

	h.payload = payload
	h.mux.Update()
}

func (h *msgHandler) hasValue() (string, error) {
	elapsed := h.mux.LockWithTimeout()
	defer h.mux.Unlock()

	if elapsed > 0 {
		return "", fmt.Errorf("%s outdated: %v", h.topic, elapsed.Truncate(time.Second))
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
