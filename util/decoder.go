package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
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

	if err == nil && validationSupported(cc) {
		err = validate.Struct(cc)
		if e, ok := err.(validator.ValidationErrors); ok {
			err = simplifyValidationErrors(e)
		}
	}

	return err
}

// simplifyValidationErrors extract simple error message for single field
func simplifyValidationErrors(errs validator.ValidationErrors) error {
	for _, e := range errs {
		switch e.Tag() {
		case "required":
			return fmt.Errorf("missing %s", strings.ToLower(e.Field()))
		case "required_with":
			return fmt.Errorf("missing %s when %s is specified", strings.ToLower(e.Field()), strings.ToLower(e.Param()))
		case "required_without":
			return fmt.Errorf("need either %s or %s", strings.ToLower(e.Field()), strings.ToLower(e.Param()))
		case "excluded_with":
			return fmt.Errorf("can only have either %s or %s", strings.ToLower(e.Field()), strings.ToLower(e.Param()))
		}
	}

	return errs
}

// validationSupported determines if type is validatable
func validationSupported(cc interface{}) bool {
	kind := reflect.TypeOf(cc).Elem().Kind()

	switch kind {
	case reflect.Map, reflect.Ptr:
		return false
	}

	return true
}
