package util

import (
	"github.com/mitchellh/mapstructure"
)

// DecodeOther uses mapstructure to decode into target structure. Unused keys cause errors.
func DecodeOther(log *Logger, other interface{}, cc interface{}) {
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
