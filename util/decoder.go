package util

import (
	"github.com/mitchellh/mapstructure"
)

// DecodeOther uses mapstructure to decode into target structure. Unused keys cause errors.
func DecodeOther(other, cc interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Result:           cc,
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err == nil {
		err = decoder.Decode(other)
	}

	if err != nil {
		err = &ConfigError{err}
	}

	return err
}

// ConfigError wraps yaml configuration errors from mapstructure
type ConfigError struct {
	err error
}

func (e *ConfigError) Error() string {
	return e.err.Error()
}

func (e *ConfigError) Unwrap() error {
	return e.err
}
