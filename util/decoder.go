package util

import (
	"fmt"
	"os"
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
)

var validate = validator.New()

// envRef matches a configuration value that references an environment variable.
// The `env:` prefix and full-value match avoid clashing with the ${var} plugin syntax.
var envRef = regexp.MustCompile(`^\$\{env:(\w+)\}$`)

// envHookFunc replaces ${env:NAME} values with the environment variable's content
func envHookFunc(f reflect.Type, _ reflect.Type, data any) (any, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}

	m := envRef.FindStringSubmatch(reflect.ValueOf(data).String())
	if m == nil {
		return data, nil
	}

	val, ok := os.LookupEnv(m[1])
	if !ok {
		return nil, fmt.Errorf("missing environment variable: %s", m[1])
	}

	return val, nil
}

// DecodeOther uses mapstructure to decode into target structure. Unused keys cause errors.
func DecodeOther(other, cc any) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Result:           cc,
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			envHookFunc,
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
