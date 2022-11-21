package profile

import (
	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
	"golang.org/x/exp/maps"
)

// OnGetConfiguration handles the CS message
func (s *Core) OnGetConfiguration(request *core.GetConfigurationRequest) (confirmation *core.GetConfigurationConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)

	var resultKeys []core.ConfigurationKey
	var unknownKeys []string

	for _, key := range request.Key {
		configKey, ok := s.configuration[key]
		if !ok {
			unknownKeys = append(unknownKeys, *configKey.Value)
		} else {
			resultKeys = append(resultKeys, configKey)
		}
	}

	// return config for all keys
	if len(request.Key) == 0 {
		resultKeys = maps.Values(s.configuration)
	}

	s.log.TRACE.Printf("%s: configuration for requested keys: %v", request.GetFeatureName(), request.Key)

	conf := core.NewGetConfigurationConfirmation(resultKeys)
	conf.UnknownKey = unknownKeys

	return conf, nil
}

// OnChangeConfiguration handles the CS message
func (s *Core) OnChangeConfiguration(request *core.ChangeConfigurationRequest) (confirmation *core.ChangeConfigurationConfirmation, err error) {
	s.log.TRACE.Printf("recv: %s %+v", request.GetFeatureName(), request)
	return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusAccepted), nil
}
