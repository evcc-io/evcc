package util

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

var validate = validator.New()

// simplifyValidationErrors extract simple error message for single field
func simplifyValidationErrors(errs validator.ValidationErrors) error {
	for _, e := range errs {
		if e.Tag() == "required" {
			return errors.New("missing " + strings.ToLower(e.Field()))
		}
	}

	return errs
}

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

		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			err = simplifyValidationErrors(validationErrors)
		}
	}

	return err
}
