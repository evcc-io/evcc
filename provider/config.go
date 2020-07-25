package provider

import (
	"strings"
	"time"

	"github.com/andig/evcc/util"
)

const (
	execTimeout = 5 * time.Second
)

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

// scriptConfig is the specific script getter/setter configuration
type scriptConfig struct {
	Cmd     string
	Timeout time.Duration
	Cache   time.Duration
}

// MQTT singleton
var MQTT *MqttClient

func mqttFromConfig(log *util.Logger, other map[string]interface{}) mqttConfig {
	if MQTT == nil {
		log.FATAL.Fatal("mqtt not configured")
	}

	var pc mqttConfig
	util.DecodeOther(log, other, &pc)

	if pc.Scale == 0 {
		pc.Scale = 1
	}

	return pc
}

func scriptFromConfig(log *util.Logger, other map[string]interface{}) scriptConfig {
	var pc scriptConfig
	util.DecodeOther(log, other, &pc)

	if pc.Timeout == 0 {
		pc.Timeout = execTimeout
	}

	return pc
}

// NewFloatGetterFromConfig creates a FloatGetter from config
func NewFloatGetterFromConfig(log *util.Logger, config Config) (res func() (float64, error)) {
	switch strings.ToLower(config.Type) {
	case "calc":
		res = NewCalcFromConfig(log, config.Other)
	case "http":
		res = NewHTTPProviderFromConfig(log, config.Other).FloatGetter
	case "websocket", "ws":
		res = NewSocketProviderFromConfig(log, config.Other).FloatGetter
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.FloatGetter(pc.Topic, pc.Scale, pc.Timeout)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		res = NewScriptProvider(pc.Timeout).FloatGetter(pc.Cmd)
		if pc.Cache > 0 {
			res = NewCached(log, res, pc.Cache).FloatGetter()
		}
	case "modbus":
		res = NewModbusFromConfig(log, config.Other).FloatGetter
	default:
		log.FATAL.Fatal("invalid plugin type:", config.Type)
	}

	return
}

// NewIntGetterFromConfig creates a IntGetter from config
func NewIntGetterFromConfig(log *util.Logger, config Config) (res func() (int64, error)) {
	switch strings.ToLower(config.Type) {
	case "http":
		res = NewHTTPProviderFromConfig(log, config.Other).IntGetter
	case "websocket", "ws":
		res = NewSocketProviderFromConfig(log, config.Other).IntGetter
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.IntGetter(pc.Topic, int64(pc.Scale), pc.Timeout)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		res = NewScriptProvider(pc.Timeout).IntGetter(pc.Cmd)
		if pc.Cache > 0 {
			res = NewCached(log, res, pc.Cache).IntGetter()
		}
	case "modbus":
		res = NewModbusFromConfig(log, config.Other).IntGetter
	default:
		log.FATAL.Fatal("invalid plugin type:", config.Type)
	}

	return
}

// NewStringGetterFromConfig creates a StringGetter from config
func NewStringGetterFromConfig(log *util.Logger, config Config) (res func() (string, error)) {
	switch strings.ToLower(config.Type) {
	case "http":
		res = NewHTTPProviderFromConfig(log, config.Other).StringGetter
	case "websocket", "ws":
		res = NewSocketProviderFromConfig(log, config.Other).StringGetter
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.StringGetter(pc.Topic, pc.Timeout)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		res = NewScriptProvider(pc.Timeout).StringGetter(pc.Cmd)
		if pc.Cache > 0 {
			res = NewCached(log, res, pc.Cache).StringGetter()
		}
	case "combined", "openwb":
		res = openWBStatusFromConfig(log, config.Other)
	default:
		log.FATAL.Fatal("invalid plugin type:", config.Type)
	}

	return
}

// NewBoolGetterFromConfig creates a BoolGetter from config
func NewBoolGetterFromConfig(log *util.Logger, config Config) (res func() (bool, error)) {
	switch strings.ToLower(config.Type) {
	case "http":
		res = NewHTTPProviderFromConfig(log, config.Other).BoolGetter
	case "websocket", "ws":
		res = NewSocketProviderFromConfig(log, config.Other).BoolGetter
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.BoolGetter(pc.Topic, pc.Timeout)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		res = NewScriptProvider(pc.Timeout).BoolGetter(pc.Cmd)
		if pc.Cache > 0 {
			res = NewCached(log, res, pc.Cache).BoolGetter()
		}
	default:
		log.FATAL.Fatal("invalid plugin type:", config.Type)
	}

	return
}

// NewIntSetterFromConfig creates a IntSetter from config
func NewIntSetterFromConfig(log *util.Logger, param string, config Config) (res func(int64) error) {
	switch strings.ToLower(config.Type) {
	case "http":
		res = NewHTTPProviderFromConfig(log, config.Other).IntSetter
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.IntSetter(param, pc.Topic, pc.Payload)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		script := NewScriptProvider(pc.Timeout)
		res = script.IntSetter(param, pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid setter type %s", config.Type)
	}
	return
}

// NewBoolSetterFromConfig creates a BoolSetter from config
func NewBoolSetterFromConfig(log *util.Logger, param string, config Config) (res func(bool) error) {
	switch strings.ToLower(config.Type) {
	case "http":
		res = NewHTTPProviderFromConfig(log, config.Other).BoolSetter
	case "mqtt":
		pc := mqttFromConfig(log, config.Other)
		res = MQTT.BoolSetter(param, pc.Topic, pc.Payload)
	case "script":
		pc := scriptFromConfig(log, config.Other)
		script := NewScriptProvider(pc.Timeout)
		res = script.BoolSetter(param, pc.Cmd)
	default:
		log.FATAL.Fatalf("invalid setter type %s", config.Type)
	}
	return
}
