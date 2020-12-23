package provider

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/andig/evcc/provider/mqtt"
	"github.com/andig/evcc/util"
)

// Mqtt provider
type Mqtt struct {
	log     *util.Logger
	client  *mqtt.Client
	topic   string
	payload string
	scale   float64
	timeout time.Duration
}

func init() {
	registry.Add("mqtt", NewMqttFromConfig)
}

// NewMqttFromConfig creates Mqtt provider
func NewMqttFromConfig(other map[string]interface{}) (IntProvider, error) {
	cc := struct {
		mqtt.Config    `mapstructure:",squash"`
		Topic, Payload string // Payload only applies to setters
		Scale          float64
		Timeout        time.Duration
	}{
		Scale: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("mqtt")

	client, err := mqtt.RegisteredClientOrDefault(log, cc.Config)
	if err != nil {
		return nil, err
	}

	m := NewMqtt(log, client, cc.Topic, cc.Payload, cc.Scale, cc.Timeout)

	return m, err
}

// NewMqtt creates mqtt provider for given topic
func NewMqtt(log *util.Logger, client *mqtt.Client, topic string, payload string, scale float64, timeout time.Duration) *Mqtt {
	m := &Mqtt{
		log:     log,
		client:  client,
		topic:   topic,
		payload: payload,
		scale:   scale,
		timeout: timeout,
	}
	return m
}

// FloatGetter creates handler for float64 from MQTT topic that returns cached value
func (m *Mqtt) FloatGetter() func() (float64, error) {
	h := &msgHandler{
		log:   m.log,
		topic: m.topic,
		scale: m.scale,
		mux:   util.NewWaiter(m.timeout, func() { m.log.TRACE.Printf("%s wait for initial value", m.topic) }),
	}

	m.client.Listen(m.topic, h.receive)
	return h.floatGetter
}

// IntGetter creates handler for int64 from MQTT topic that returns cached value
func (m *Mqtt) IntGetter() func() (int64, error) {
	h := &msgHandler{
		log:   m.log,
		topic: m.topic,
		scale: float64(m.scale),
		mux:   util.NewWaiter(m.timeout, func() { m.log.TRACE.Printf("%s wait for initial value", m.topic) }),
	}

	m.client.Listen(m.topic, h.receive)
	return h.intGetter
}

// StringGetter creates handler for string from MQTT topic that returns cached value
func (m *Mqtt) StringGetter() func() (string, error) {
	h := &msgHandler{
		log:   m.log,
		topic: m.topic,
		mux:   util.NewWaiter(m.timeout, func() { m.log.TRACE.Printf("%s wait for initial value", m.topic) }),
	}

	m.client.Listen(m.topic, h.receive)
	return h.stringGetter
}

// BoolGetter creates handler for string from MQTT topic that returns cached value
func (m *Mqtt) BoolGetter() func() (bool, error) {
	h := &msgHandler{
		log:   m.log,
		topic: m.topic,
		mux:   util.NewWaiter(m.timeout, func() { m.log.TRACE.Printf("%s wait for initial value", m.topic) }),
	}

	m.client.Listen(m.topic, h.receive)
	return h.boolGetter
}

// IntSetter publishes topic with parameter replaced by int value
func (m *Mqtt) IntSetter(param string) func(int64) error {
	return func(v int64) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		m.log.TRACE.Printf("send %s: '%s'", m.topic, payload)
		return m.client.Publish(m.topic, false, payload)
	}
}

// BoolSetter invokes script with parameter replaced by bool value
func (m *Mqtt) BoolSetter(param string) func(bool) error {
	return func(v bool) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		m.log.TRACE.Printf("send %s: '%s'", m.topic, payload)
		return m.client.Publish(m.topic, false, payload)
	}
}

type msgHandler struct {
	log     *util.Logger
	mux     *util.Waiter
	scale   float64
	topic   string
	payload string
}

func (h *msgHandler) receive(payload string) {
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
