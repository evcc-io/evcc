package util

import (
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
)

var validate = validator.New()

// Validate validates a single value against a validator tag, e.g. "gte=0"
func Validate(v any, tag string) error {
	return validate.Var(v, tag)
}

// DecodeOther uses mapstructure to decode into target structure. Unused keys cause errors.
func DecodeOther(other, cc any) error {
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
	if err != nil {
		return err
	}

	if err := decoder.Decode(other); err != nil {
		return &ConfigError{err}
	}

	// validate structs
	if rv := reflect.ValueOf(cc); rv.Kind() == reflect.Struct || rv.Kind() == reflect.Pointer && rv.Elem().Kind() == reflect.Struct {
		return validate.Struct(cc)
	}

	return nil
}

// ConfigError wraps yaml configuration errors from mapstructure
type ConfigError struct {
	err error
}

func NewConfigError(err error) error {
	return &ConfigError{err}
}

func (e ConfigError) Error() string {
	return e.err.Error()
}

func (e ConfigError) Unwrap() error {
	return e.err
}
