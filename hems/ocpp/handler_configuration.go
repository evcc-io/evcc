package ocpp

import "github.com/lorenzodonini/ocpp-go/ocpp1.6/core"

func (s *OCPP) OnGetConfiguration(request *core.GetConfigurationRequest) (confirmation *core.GetConfigurationConfirmation, err error) {
	var resultKeys []core.ConfigurationKey
	var unknownKeys []string

	// for _, key := range request.Key {
	// 	configKey, ok := s.configuration[key]
	// 	if !ok {
	// 		unknownKeys = append(unknownKeys, configKey.Value)
	// 	} else {
	// 		resultKeys = append(resultKeys, configKey)
	// 	}
	// }

	// if len(request.Key) == 0 {
	// 	// Return config for all keys
	// 	for _, v := range s.configuration {
	// 		resultKeys = append(resultKeys, v)
	// 	}
	// }

	s.log.TRACE.Printf("%s: returning configuration for requested keys: %v", request.GetFeatureName(), request.Key)

	conf := core.NewGetConfigurationConfirmation(resultKeys)
	conf.UnknownKey = unknownKeys

	return conf, nil
}

func (s *OCPP) OnChangeConfiguration(request *core.ChangeConfigurationRequest) (confirmation *core.ChangeConfigurationConfirmation, err error) {
	configKey, ok := s.configuration[request.Key]

	if !ok {
		s.log.TRACE.Printf("%s: couldn't change configuration for unsupported parameter %v", request.GetFeatureName(), configKey.Key)
		return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusNotSupported), nil
	} else if configKey.Readonly {
		s.log.TRACE.Printf("%s: couldn't change configuration for readonly parameter %v", request.GetFeatureName(), configKey.Key)
		return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusRejected), nil
	}

	configKey.Value = request.Value
	s.configuration[request.Key] = configKey

	s.log.TRACE.Printf("%s: changed configuration for parameter %v to %v", request.GetFeatureName(), configKey.Key, configKey.Value)

	return core.NewChangeConfigurationConfirmation(core.ConfigurationStatusAccepted), nil
}
