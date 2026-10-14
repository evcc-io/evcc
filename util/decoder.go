package util

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
)

var validate = validator.New()

// Validate validates a single value against a validator tag, e.g. "gte=0".
// The validator's message names no field for a plain value, so report value and rule instead.
func Validate(v any, tag string) error {
	var errs validator.ValidationErrors
	if err := validate.Var(v, tag); errors.As(err, &errs) && len(errs) > 0 {
		return fmt.Errorf("%v violates %s=%s", v, errs[0].Tag(), errs[0].Param())
	} else if err != nil {
		return err
	}
	return nil
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
