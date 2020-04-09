package provider

import (
	"time"
)

const (
	execTimeout = 5 * time.Second
	mqttTimeout = 5 * time.Second
)

// Config contains the getter/setter configuration
type Config struct {
	Type       string
	Topic      string
	Cmd        string
	Multiplier float64
	Timeout    time.Duration
	Cache      time.Duration
}

// MQTT singleton
var MQTT *MqttClient

// NewStringGetterFromConfig creates a StringGetter from config
func NewStringGetterFromConfig(pc *Config) (res StringGetter) {
	switch pc.Type {
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := NewScriptProvider(pc.Timeout)
		res = exec.StringGetter(pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid provider type %s", pc.Type)
	}

	if pc.Cache > 0 {
		res = NewCached(res, pc.Cache).StringGetter()
	}

	return
}

// NewBoolGetterFromConfig creates a BoolGetter from config
func NewBoolGetterFromConfig(pc *Config) (res BoolGetter) {
	switch pc.Type {
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := NewScriptProvider(pc.Timeout)
		res = exec.BoolGetter(pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid provider type %s", pc.Type)
	}

	if pc.Cache > 0 {
		res = NewCached(res, pc.Cache).BoolGetter()
	}

	return
}

// NewFloatGetterFromConfig creates a FloatGetter from config
func NewFloatGetterFromConfig(pc *Config) (res FloatGetter) {
	switch pc.Type {
	case "mqtt":
		if MQTT == nil {
			log.FATAL.Fatal("mqtt not configured")
		}
		if pc.Timeout == 0 {
			pc.Timeout = mqttTimeout
		}
		if pc.Multiplier == 0 {
			pc.Multiplier = 1
		}
		res = MQTT.FloatGetter(pc.Topic, pc.Multiplier, pc.Timeout)
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := NewScriptProvider(pc.Timeout)
		res = exec.FloatGetter(pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid provider type %s", pc.Type)
	}

	if pc.Cache > 0 {
		res = NewCached(res, pc.Cache).FloatGetter()
	}

	return
}

// NewBoolSetterFromConfig creates a BoolSetter from config
func NewBoolSetterFromConfig(param string, pc *Config) (res BoolSetter) {
	switch pc.Type {
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := NewScriptProvider(pc.Timeout)
		res = exec.BoolSetter(param, pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid setter type %s", pc.Type)
	}
	return
}

// NewIntSetterFromConfig creates a IntSetter from config
func NewIntSetterFromConfig(param string, pc *Config) (res IntSetter) {
	switch pc.Type {
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := NewScriptProvider(pc.Timeout)
		res = exec.IntSetter(param, pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid setter type %s", pc.Type)
	}
	return
}
