package util

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

var validate = validator.New()

// simplifyValidationErrors extract simple error message for single field
func simplifyValidationErrors(errs validator.ValidationErrors) error {
	for _, e := range errs {
		switch e.Tag() {
		case "required":
			return fmt.Errorf("missing %s", strings.ToLower(e.Field()))
		case "required_without":
			return fmt.Errorf("need either %s or %s", strings.ToLower(e.Field()), strings.ToLower(e.Param()))
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
		if err != nil {
			err = simplifyValidationErrors(err.(validator.ValidationErrors))
		}
	}

	return err
}
