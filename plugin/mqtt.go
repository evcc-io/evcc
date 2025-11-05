package plugin

import (
	"context"
	"time"

	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/plugin/pipeline"
	"github.com/evcc-io/evcc/util"
)

// Mqtt provider
type Mqtt struct {
	*getter
	log      *util.Logger
	client   *mqtt.Client
	topic    string
	retained bool
	payload  string
	timeout  time.Duration
	pipeline *pipeline.Pipeline
}

func init() {
	registry.AddCtx("mqtt", NewMqttPluginFromConfig)
}

// NewMqttPluginFromConfig creates Mqtt provider
func NewMqttPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
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

	log := contextLogger(ctx, util.NewLogger("mqtt"))

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
		timeout: timeout,
	}

	m.getter = defaultGetters(m, 1)

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

// newReceiver creates a msgHandler and subscribes it to the topic.
func (m *Mqtt) newReceiver() (*msgHandler, error) {
	h := &msgHandler{
		topic:    m.topic,
		pipeline: m.pipeline,
		val:      util.NewMonitor[string](m.timeout),
	}

	err := m.client.Listen(m.topic, h.receive)
	return h, err
}

var _ Getters = (*Mqtt)(nil)

// StringGetter creates handler for string from MQTT topic that returns cached value
func (m *Mqtt) StringGetter() (func() (string, error), error) {
	h, err := m.newReceiver()
	return h.value, err
}

var _ IntSetter = (*Mqtt)(nil)

// IntSetter publishes topic with parameter replaced by int value
func (m *Mqtt) IntSetter(param string) (func(int64) error, error) {
	return func(v int64) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		m.client.Publish(m.topic, m.retained, payload)
		return nil
	}, nil
}

var _ FloatSetter = (*Mqtt)(nil)

// FloatSetter publishes topic with parameter replaced by float value
func (m *Mqtt) FloatSetter(param string) (func(float64) error, error) {
	return func(v float64) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		m.client.Publish(m.topic, m.retained, payload)
		return nil
	}, nil
}

var _ BoolSetter = (*Mqtt)(nil)

// BoolSetter invokes script with parameter replaced by bool value
func (m *Mqtt) BoolSetter(param string) (func(bool) error, error) {
	return func(v bool) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		m.client.Publish(m.topic, m.retained, payload)
		return nil
	}, nil
}

var _ StringSetter = (*Mqtt)(nil)

// StringSetter invokes script with parameter replaced by string value
func (m *Mqtt) StringSetter(param string) (func(string) error, error) {
	return func(v string) error {
		payload, err := setFormattedValue(m.payload, param, v)
		if err != nil {
			return err
		}

		m.client.Publish(m.topic, m.retained, payload)
		return nil
	}, nil
}
