package provider

import (
	"strings"
	"time"

	"github.com/andig/evcc/api"
)

const (
	execTimeout = 5 * time.Second
	mqttTimeout = 5 * time.Second
)

// Config is the general provider config
type Config struct {
	Type  string
	Other map[string]interface{} `mapstructure:",remain"`
}

// mqttConfig is the specific mqtt getter/setter configuration
type mqttConfig struct {
	Topic, Payload string // Payload only applies to setters
	Multiplier float64
	Timeout    time.Duration
}

// scriptConfig is the specific script getter/setter configuration
type scriptConfig struct {
	Cmd     string
	Timeout time.Duration
	Cache   time.Duration
}

// MQTT singleton
var MQTT *MqttClient

func mqttFromConfig(log *api.Logger, other map[string]interface{}) mqttConfig {
	if MQTT == nil {
		log.FATAL.Fatal("mqtt not configured")
	}

	var pc mqttConfig
	api.DecodeOther(log, other, &pc)

	if pc.Timeout == 0 {
		pc.Timeout = mqttTimeout
	}
	if pc.Multiplier == 0 {
		pc.Multiplier = 1
	}

	return pc
}

func scriptFromConfig(log *api.Logger, other map[string]interface{}) scriptConfig {
	var pc scriptConfig
	api.DecodeOther(log, other, &pc)

	if pc.Timeout == 0 {
		pc.Timeout = execTimeout
	}

	return pc
}

// NewFloatGetterFromConfig creates a FloatGetter from config
func NewFloatGetterFromConfig(log *api.Logger, config Config) (res FloatGetter) {
	switch strings.ToLower(config.Type) {
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.FloatGetter(pc.Topic, pc.Multiplier, pc.Timeout)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		res = NewScriptProvider(pc.Timeout).FloatGetter(pc.Cmd)
		if pc.Cache > 0 {
			res = NewCached(res, pc.Cache).FloatGetter()
		}
	default:
		log.FATAL.Fatalf("invalid provider type %s", config.Type)
	}

	return
}

// NewIntGetterFromConfig creates a IntGetter from config
func NewIntGetterFromConfig(log *api.Logger, config Config) (res IntGetter) {
	switch strings.ToLower(config.Type) {
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.IntGetter(pc.Topic, int64(pc.Multiplier), pc.Timeout)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		res = NewScriptProvider(pc.Timeout).IntGetter(pc.Cmd)
		if pc.Cache > 0 {
			res = NewCached(res, pc.Cache).IntGetter()
		}
	default:
		log.FATAL.Fatalf("invalid provider type %s", config.Type)
	}

	return
}

// NewStringGetterFromConfig creates a StringGetter from config
func NewStringGetterFromConfig(log *api.Logger, config Config) (res StringGetter) {
	switch strings.ToLower(config.Type) {
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.StringGetter(pc.Topic, pc.Timeout)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		res = NewScriptProvider(pc.Timeout).StringGetter(pc.Cmd)
		if pc.Cache > 0 {
			res = NewCached(res, pc.Cache).StringGetter()
		}
	default:
		log.FATAL.Fatalf("invalid provider type %s", config.Type)
	}

	return
}

// NewBoolGetterFromConfig creates a BoolGetter from config
func NewBoolGetterFromConfig(log *api.Logger, config Config) (res BoolGetter) {
	switch strings.ToLower(config.Type) {
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.BoolGetter(pc.Topic, pc.Timeout)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		res = NewScriptProvider(pc.Timeout).BoolGetter(pc.Cmd)
		if pc.Cache > 0 {
			res = NewCached(res, pc.Cache).BoolGetter()
		}
	default:
		log.FATAL.Fatalf("invalid provider type %s", config.Type)
	}

	return
}

// NewIntSetterFromConfig creates a IntSetter from config
func NewIntSetterFromConfig(log *api.Logger, param string, config Config) (res IntSetter) {
	switch strings.ToLower(config.Type) {
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.IntSetter(param, pc.Topic, pc.Payload)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		exec := NewScriptProvider(pc.Timeout)
		res = exec.IntSetter(param, pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid setter type %s", config.Type)
	}
	return
}

// NewBoolSetterFromConfig creates a BoolSetter from config
func NewBoolSetterFromConfig(log *api.Logger, param string, config Config) (res BoolSetter) {
	switch strings.ToLower(config.Type) {
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.BoolSetter(param, pc.Topic, pc.Payload)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		exec := NewScriptProvider(pc.Timeout)
		res = exec.BoolSetter(param, pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid setter type %s", config.Type)
	}
	return
}
