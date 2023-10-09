package provider

import (
	"time"

	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/provider/pipeline"
	"github.com/evcc-io/evcc/util"
)

// Mqtt provider
type Mqtt struct {
	log      *util.Logger
	client   *mqtt.Client
	topic    string
	retained bool
	payload  string
	scale    float64
	timeout  time.Duration
	pipeline *pipeline.Pipeline
}

func init() {
	registry.Add("mqtt", NewMqttFromConfig)
}

// NewMqttFromConfig creates Mqtt provider
func NewMqttFromConfig(other map[string]interface{}) (Provider, error) {
	cc := struct {
		mqtt.Config       `mapstructure:",squash"`
		Topic, Payload    string // Payload only applies to setters
		Retained          bool
		Scale             float64
		Timeout           time.Duration
		pipeline.Settings `mapstructure:",squash"`
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

	m := NewMqtt(log, client, cc.Topic, cc.Timeout).WithScale(cc.Scale).WithPayload(cc.Payload)
	if cc.Retained {
		m = m.WithRetained()
	}

	pipe, err := pipeline.New(log, cc.Settings)
	if err == nil {
		m = m.WithPipeline(pipe)
	}

	return m, err
}

// NewMqtt creates mqtt provider for given topic
func NewMqtt(log *util.Logger, client *mqtt.Client, topic string, timeout time.Duration) *Mqtt {
	m := &Mqtt{
		log:     log,
		client:  client,
		topic:   topic,
		scale:   1,
		timeout: timeout,
	}

	return m
}

// WithPayload adds payload for setters
func (m *Mqtt) WithPayload(payload string) *Mqtt {
	m.payload = payload
	return m
}

// WithRetained adds retained flag for setters
func (m *Mqtt) WithRetained() *Mqtt {
	m.retained = true
	return m
}

// WithScale sets scaler for getters
func (m *Mqtt) WithScale(scale float64) *Mqtt {
	m.scale = scale
	return m
}

// WithPipeline adds a processing pipeline
func (p *Mqtt) WithPipeline(pipeline *pipeline.Pipeline) *Mqtt {
	p.pipeline = pipeline
	return p
}

var _ FloatProvider = (*Mqtt)(nil)

// newReceiver creates a msgHandler and subscribes it to the topic.
// receiver will ensure actual data guarded by `timeout` and return error
// if initial value is not received within `timeout` or max. 10s if timeout is not given.
func (m *Mqtt) newReceiver() *msgHandler {
	h := &msgHandler{
		topic:    m.topic,
		scale:    m.scale,
		wait:     util.NewWaiter(m.timeout, func() { m.log.DEBUG.Printf("%s wait for initial value", m.topic) }),
		pipeline: m.pipeline,
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

		return m.client.Publish(m.topic, m.retained, payload)
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

		return m.client.Publish(m.topic, m.retained, payload)
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

		return m.client.Publish(m.topic, m.retained, payload)
	}
}
