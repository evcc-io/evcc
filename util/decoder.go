package util

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/mitchellh/mapstructure"
)

var (
	validate = validator.New()
	trans    ut.Translator
)

func init() {
	en := en.New()
	uni := ut.New(en, en)

	trans, _ = uni.GetTranslator("en")
	_ = en_translations.RegisterDefaultTranslations(validate, trans)

	// simplify required field error
	if err := validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "missing {0}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", strings.ToLower(fe.Field()))
		return t
	}); err != nil {
		panic(err)
	}
}

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
	if err != nil {
		return err
	}

	if err := decoder.Decode(other); err != nil {
		return &ConfigError{err}
	}

	// validate structs
	if rv := reflect.ValueOf(cc); rv.Kind() == reflect.Struct || rv.Kind() == reflect.Pointer && rv.Elem().Kind() == reflect.Struct {
		err := validate.Struct(cc)

		// translate validation errors
		if verrs, ok := err.(validator.ValidationErrors); ok {
			errs := make([]error, 0, len(verrs))
			for _, e := range verrs {
				errs = append(errs, errors.New(e.Translate(trans)))
			}
			return errors.Join(errs...)
		}

		return err
	}

	return nil
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
