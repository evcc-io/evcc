package util

import (
	"github.com/mitchellh/mapstructure"
	"gopkg.in/go-playground/validator.v9"
)

var validate = validator.New()

// DecodeOther uses mapstructure to decode into target structure. Unused keys cause errors.
func DecodeOther(other interface{}, cc interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Result:           cc,
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err == nil {
		err = decoder.Decode(other)
	}

	if err == nil {
		err = validate.Struct(cc)
	}

	return err
}
