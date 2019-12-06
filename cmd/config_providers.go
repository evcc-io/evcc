package cmd

import (
	"time"

	"github.com/andig/evcc/provider"
)

const (
	execTimeout = 5 * time.Second
	mqttTimeout = 5 * time.Second
)

func stringGetter(pc *providerConfig) (res provider.StringGetter) {
	switch pc.Type {
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := provider.NewScriptProvider(pc.Timeout)
		res = exec.StringGetter(pc.Cmd)
	default:
		log.FATAL.Printf("invalid provider type %s", pc.Type)
	}

	if pc.Cache > 0 {
		res = provider.NewCacheGetter(res, pc.Cache).StringGetter
	}

	return
}

func boolGetter(pc *providerConfig) (res provider.BoolGetter) {
	switch pc.Type {
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := provider.NewScriptProvider(pc.Timeout)
		res = exec.BoolGetter(pc.Cmd)
	default:
		log.FATAL.Printf("invalid provider type %s", pc.Type)
	}

	if pc.Cache > 0 {
		res = provider.NewCacheGetter(res, pc.Cache).BoolGetter
	}

	return
}

func intProvider(pc *providerConfig) (res provider.IntGetter) {
	switch pc.Type {
	case "mqtt":
		if pc.Timeout == 0 {
			pc.Timeout = mqttTimeout
		}
		res = mq.IntGetter(pc.Topic, pc.Timeout)
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := provider.NewScriptProvider(pc.Timeout)
		res = exec.IntGetter(pc.Cmd)
	default:
		log.FATAL.Printf("invalid provider type %s", pc.Type)
	}

	if pc.Cache > 0 {
		res = provider.NewCacheGetter(res, pc.Cache).IntGetter
	}

	return
}

func floatGetter(pc *providerConfig) (res provider.FloatGetter) {
	switch pc.Type {
	case "mqtt":
		if pc.Timeout == 0 {
			pc.Timeout = mqttTimeout
		}
		res = mq.FloatGetter(pc.Topic, pc.Timeout)
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := provider.NewScriptProvider(pc.Timeout)
		res = exec.FloatGetter(pc.Cmd)
	default:
		log.FATAL.Printf("invalid provider type %s", pc.Type)
	}

	if pc.Cache > 0 {
		res = provider.NewCacheGetter(res, pc.Cache).FloatGetter
	}

	return
}

func boolSetter(param string, pc *providerConfig) (res provider.BoolSetter) {
	switch pc.Type {
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := provider.NewScriptProvider(pc.Timeout)
		res = exec.BoolSetter(param, pc.Cmd)
	default:
		log.FATAL.Printf("invalid setter type %s", pc.Type)
	}
	return
}

func intSetter(param string, pc *providerConfig) (res provider.IntSetter) {
	switch pc.Type {
	case "script":
		if pc.Timeout == 0 {
			pc.Timeout = execTimeout
		}
		exec := provider.NewScriptProvider(pc.Timeout)
		res = exec.IntSetter(param, pc.Cmd)
	default:
		log.FATAL.Printf("invalid setter type %s", pc.Type)
	}
	return
}
