package provider

import (
	"fmt"
	"regexp"
	"time"

	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/itchyny/gojq"
)

// Mqtt provider
type Mqtt struct {
	log     *util.Logger
	client  *mqtt.Client
	topic   string
	payload string
	scale   float64
	timeout time.Duration
	re      *regexp.Regexp
	jq      *gojq.Query
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
		Regex          string
		Jq             string
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

	m := NewMqtt(log, client, cc.Topic, cc.Scale, cc.Timeout).WithPayload(cc.Payload)

	if cc.Regex != "" {
		if m, err = m.WithRegex(cc.Regex); err != nil {
			return nil, err
		}
	}

	if cc.Jq != "" {
		if m, err = m.WithJq(cc.Jq); err != nil {
			return nil, err
		}
	}

	return m, nil
}

// NewMqtt creates mqtt provider for given topic
func NewMqtt(log *util.Logger, client *mqtt.Client, topic string, scale float64, timeout time.Duration) *Mqtt {
	m := &Mqtt{
		log:     log,
		client:  client,
		topic:   topic,
		scale:   scale,
		timeout: timeout,
	}

	return m
}

// WithPayload adds payload for setters
func (m *Mqtt) WithPayload(payload string) *Mqtt {
	m.payload = payload
	return m
}

// WithRegex adds a regex query applied to the mqtt listener payload
func (m *Mqtt) WithRegex(regex string) (*Mqtt, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return m, fmt.Errorf("invalid regex '%s': %w", re, err)
	}

	m.re = re

	return m, nil
}

// WithJq adds a jq query applied to the mqtt listener payload
func (m *Mqtt) WithJq(jq string) (*Mqtt, error) {
	op, err := gojq.Parse(jq)
	if err != nil {
		return m, fmt.Errorf("invalid jq query '%s': %w", jq, err)
	}

	m.jq = op

	return m, nil
}

var _ FloatProvider = (*Mqtt)(nil)

// newReceiver creates a msgHandler and subscribes it to the topic.
// receiver will ensure actual data guarded by `timeout` and return error
// if initial value is not received within `timeout` or max. 10s if timeout is not given.
func (m *Mqtt) newReceiver() *msgHandler {
	wait := m.timeout
	if wait == 0 {
		wait = request.Timeout
	}

	h := &msgHandler{
		topic: m.topic,
		scale: m.scale,
		mux:   util.NewWaiter(wait, func() { m.log.DEBUG.Printf("%s wait for initial value", m.topic) }),
		re:    m.re,
		jq:    m.jq,
	}

	m.client.Listen(m.topic, h.receive)
	return h
}

// FloatGetter creates handler for float64 from MQTT topic that returns cached value
func (m *Mqtt) FloatGetter() func() (float64, error) {
	h := m.newReceiver()
	return h.floatGetter
}

var _ IntProvider = (*Mqtt)(nil)

// IntGetter creates handler for int64 from MQTT topic that returns cached value
func (m *Mqtt) IntGetter() func() (int64, error) {
	h := m.newReceiver()
	return h.intGetter
}

var _ StringProvider = (*Mqtt)(nil)

// StringGetter creates handler for string from MQTT topic that returns cached value
func (m *Mqtt) StringGetter() func() (string, error) {
	h := m.newReceiver()
	return h.stringGetter
}

var _ BoolProvider = (*Mqtt)(nil)

// BoolGetter creates handler for string from MQTT topic that returns cached value
func (m *Mqtt) BoolGetter() func() (bool, error) {
	h := m.newReceiver()
	return h.boolGetter
}

var _ SetIntProvider = (*Mqtt)(nil)

// IntSetter publishes topic with parameter replaced by int value
func (m *Mqtt) IntSetter(param string) func(int64) error {
	return func(v int64) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		return m.client.Publish(m.topic, false, payload)
	}
}

var _ SetBoolProvider = (*Mqtt)(nil)

// BoolSetter invokes script with parameter replaced by bool value
func (m *Mqtt) BoolSetter(param string) func(bool) error {
	return func(v bool) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		return m.client.Publish(m.topic, false, payload)
	}
}

var _ SetStringProvider = (*Mqtt)(nil)

// StringSetter invokes script with parameter replaced by string value
func (m *Mqtt) StringSetter(param string) func(string) error {
	return func(v string) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		return m.client.Publish(m.topic, false, payload)
	}
}
