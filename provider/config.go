package provider

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/util"
)

type Provider interface {
	IntGetter() (int64, error)
}

type StringProvider interface {
	StringGetter() (string, error)
}

type FloatProvider interface {
	FloatGetter() (float64, error)
}

type BoolProvider interface {
	BoolGetter() (bool, error)
}

type SetProvider interface {
	IntSetter(param string) func(int64) error
}

type SetBoolProvider interface {
	BoolSetter(param string) func(bool) error
}

type providerRegistry map[string]func(map[string]interface{}) (Provider, error)

func (r providerRegistry) Add(name string, factory func(map[string]interface{}) (Provider, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate plugin type: %s", name))
	}
	r[name] = factory
}

func (r providerRegistry) Get(name string) (func(map[string]interface{}) (Provider, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("plugin type not registered: %s", name)
	}
	return factory, nil
}

var registry providerRegistry = make(map[string]func(map[string]interface{}) (Provider, error))

// newFromConfig creates plugin from configuration
func newFromConfig(typ string, other map[string]interface{}) (v Provider, err error) {
	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create type '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid plugin type: %s", typ)
	}

	return
}

// Config is the general provider config
type Config struct {
	Type  string
	Other map[string]interface{} `mapstructure:",remain"`
}

// mqttConfig is the specific mqtt getter/setter configuration
type mqttConfig struct {
	Topic, Payload string // Payload only applies to setters
	Scale          float64
	Timeout        time.Duration
}

// MQTT singleton
var MQTT *MqttClient

func mqttFromConfig(other map[string]interface{}) (mqttConfig, error) {
	pc := mqttConfig{Scale: 1}
	if err := util.DecodeOther(other, &pc); err != nil {
		return pc, err
	}

	if MQTT == nil {
		return pc, errors.New("mqtt not configured")
	}

	return pc, nil
}

// NewFloatGetterFromConfig creates a FloatGetter from config
func NewFloatGetterFromConfig(config Config) (res func() (float64, error), err error) {
	switch typ := strings.ToLower(config.Type); typ {
	case "calc":
		res, err = NewCalcFromConfig(config.Other)
	case "mqtt":
		if pc, err := mqttFromConfig(config.Other); err == nil {
			res = MQTT.FloatGetter(pc.Topic, pc.Scale, pc.Timeout)
		}

	default:
		factory, err := registry.Get(typ)
		if err == nil {
			var provider Provider
			provider, err = factory(config.Other)

			if prov, ok := provider.(FloatProvider); ok {
				res = prov.FloatGetter
			}
		}

		if err == nil && res == nil {
			err = fmt.Errorf("invalid plugin type: %s", config.Type)
		}
	}

	return
}

// NewIntGetterFromConfig creates a IntGetter from config
func NewIntGetterFromConfig(config Config) (res func() (int64, error), err error) {
	switch typ := strings.ToLower(config.Type); typ {
	case "mqtt":
		var pc mqttConfig
		if pc, err = mqttFromConfig(config.Other); err == nil {
			res = MQTT.IntGetter(pc.Topic, int64(pc.Scale), pc.Timeout)
		}

	default:
		factory, err := registry.Get(typ)
		if err == nil {
			var provider Provider
			provider, err = factory(config.Other)

			if err == nil {
				res = provider.IntGetter
			}
		}

		if err == nil && res == nil {
			err = fmt.Errorf("invalid plugin type: %s", config.Type)
		}
	}

	return
}

// NewStringGetterFromConfig creates a StringGetter from config
func NewStringGetterFromConfig(config Config) (res func() (string, error), err error) {
	switch typ := strings.ToLower(config.Type); typ {
	case "mqtt":
		var pc mqttConfig
		if pc, err = mqttFromConfig(config.Other); err == nil {
			res = MQTT.StringGetter(pc.Topic, pc.Timeout)
		}
	case "combined", "openwb":
		res, err = NewOpenWBStatusProviderFromConfig(config.Other)

	default:
		factory, err := registry.Get(typ)
		if err == nil {
			var provider Provider
			provider, err = factory(config.Other)

			if prov, ok := provider.(StringProvider); ok {
				res = prov.StringGetter
			}
		}

		if err == nil && res == nil {
			err = fmt.Errorf("invalid plugin type: %s", config.Type)
		}
	}

	return
}

// NewBoolGetterFromConfig creates a BoolGetter from config
func NewBoolGetterFromConfig(config Config) (res func() (bool, error), err error) {
	switch typ := strings.ToLower(config.Type); typ {
	case "mqtt":
		var pc mqttConfig
		if pc, err = mqttFromConfig(config.Other); err == nil {
			res = MQTT.BoolGetter(pc.Topic, pc.Timeout)
		}

	default:
		factory, err := registry.Get(typ)
		if err == nil {
			var provider Provider
			provider, err = factory(config.Other)

			if prov, ok := provider.(BoolProvider); ok {
				res = prov.BoolGetter
			}
		}

		if err == nil && res == nil {
			err = fmt.Errorf("invalid plugin type: %s", config.Type)
		}
	}

	return
}

// NewIntSetterFromConfig creates a IntSetter from config
func NewIntSetterFromConfig(param string, config Config) (res func(int64) error, err error) {
	switch typ := strings.ToLower(config.Type); typ {
	case "mqtt":
		var pc mqttConfig
		if pc, err = mqttFromConfig(config.Other); err == nil {
			res = MQTT.IntSetter(param, pc.Topic, pc.Payload)
		}

	default:
		factory, err := registry.Get(typ)
		if err == nil {
			var provider Provider
			provider, err = factory(config.Other)

			if prov, ok := provider.(SetProvider); ok {
				res = prov.IntSetter(param)
			}
		}

		if err == nil && res == nil {
			err = fmt.Errorf("invalid plugin type: %s", config.Type)
		}
	}

	return
}

// NewBoolSetterFromConfig creates a BoolSetter from config
func NewBoolSetterFromConfig(param string, config Config) (res func(bool) error, err error) {
	switch typ := strings.ToLower(config.Type); typ {
	case "mqtt":
		var pc mqttConfig
		if pc, err = mqttFromConfig(config.Other); err == nil {
			res = MQTT.BoolSetter(param, pc.Topic, pc.Payload)
		}

	default:
		factory, err := registry.Get(typ)
		if err == nil {
			var provider Provider
			provider, err = factory(config.Other)

			if prov, ok := provider.(SetBoolProvider); ok {
				res = prov.BoolSetter(param)
			}
		}

		if err == nil && res == nil {
			err = fmt.Errorf("invalid plugin type: %s", config.Type)
		}
	}

	return
}
