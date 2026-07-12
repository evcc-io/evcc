package export

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util/locale"
	"github.com/fatih/structs"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// normalizeValue converts a field into a typed cell value: nil for empty, floats rounded to digits, times localized
func normalizeValue(value any, digits int) any {
	if rv := reflect.ValueOf(value); rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		value = rv.Elem().Interface()
	}

	switch v := value.(type) {
	case float64:
		p := math.Pow10(digits)
		return math.Round(v*p) / p
	case time.Time:
		if v.IsZero() {
			return nil
		}
		return v.Local()
	default:
		return v
	}
}

func writeHeader(ww RowWriter, structType any, i18nPrefix string) error {
	localizer := ww.Localizer()

	var row []any
	for _, f := range structs.Fields(structType) {
		csvTag := f.Tag("csv")
		if csvTag == "-" {
			continue
		}

		caption, err := localizer.Localize(&locale.Config{
			MessageID: i18nPrefix + "." + strings.ToLower(f.Name()),
		})
		if err != nil {
			if csvTag != "" {
				caption = csvTag
			} else {
				caption = f.Name()
			}
		}

		row = append(row, caption)
	}

	return ww.Write(row)
}

func writeRow(ww RowWriter, structVal any) error {
	var row []any
	for _, f := range structs.Fields(structVal) {
		if f.Tag("csv") == "-" {
			continue
		}

		digits := 4
		if format := f.Tag("format"); format == "int" {
			digits = 0
		}

		row = append(row, normalizeValue(f.Value(), digits))
	}

	return ww.Write(row)
}

// Localizer returns an i18n localizer for the context's locale.
func Localizer(ctx context.Context) *i18n.Localizer {
	if val, ok := ctx.Value(locale.Locale).(string); ok && val != "" {
		return i18n.NewLocalizer(locale.Bundle, val, locale.Language)
	}
	return locale.Localizer
}

// WriteStructSlice emits a slice of structs to ww with localized headers.
func WriteStructSlice(ww RowWriter, slice any, cfg Config) error {
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() == reflect.Pointer {
		sliceVal = sliceVal.Elem()
	}
	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice, got %T", slice)
	}

	var structType any
	if sliceVal.Len() > 0 {
		structType = sliceVal.Index(0).Interface()
	} else {
		structType = reflect.New(sliceVal.Type().Elem()).Elem().Interface()
	}

	if err := writeHeader(ww, structType, cfg.I18nPrefix); err != nil {
		return err
	}

	for i := 0; i < sliceVal.Len(); i++ {
		if err := writeRow(ww, sliceVal.Index(i).Interface()); err != nil {
			return err
		}
	}

	ww.Flush()
	return ww.Error()
}
