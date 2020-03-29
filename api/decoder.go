package api

import (
	"github.com/mitchellh/mapstructure"
)

// DecodeOther decodes string map into target configuration
func DecodeOther(log *Logger, other map[string]interface{}, cc interface{}) {
	decoderConfig := &mapstructure.DecoderConfig{
		Result:           cc,
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	if err := decoder.Decode(other); err != nil {
		log.FATAL.Fatal(err)
	}
}
