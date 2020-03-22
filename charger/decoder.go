package charger

import (
	"github.com/andig/evcc/api"
	"github.com/mitchellh/mapstructure"
)

func decodeOther(log *api.Logger, other map[string]interface{}, cc interface{}) {
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
