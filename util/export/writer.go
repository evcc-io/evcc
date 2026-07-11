package export

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util/locale"
	"github.com/fatih/structs"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

func formatValue(mp *message.Printer, value any, digits int) string {
	if rv := reflect.ValueOf(value); rv.Kind() == reflect.Pointer && rv.IsNil() {
		return ""
	}

	switch v := value.(type) {
	case float64:
		return mp.Sprint(number.Decimal(v, number.NoSeparator(), number.MaxFractionDigits(digits)))
	case *float64:
		return mp.Sprint(number.Decimal(*v, number.NoSeparator(), number.MaxFractionDigits(digits)))
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.Local().Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", value)
	}
}

func writeHeader(ww RowWriter, structType any, i18nPrefix string) error {
	localizer := ww.Localizer()

	var row []string
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
	mp := ww.Printer()

	var row []string
	for _, f := range structs.Fields(structVal) {
		if f.Tag("csv") == "-" {
			continue
		}

		digits := 4
		if format := f.Tag("format"); format == "int" {
			digits = 0
		}

		row = append(row, formatValue(mp, f.Value(), digits))
	}

	return ww.Write(row)
}

// LocaleTag resolves the export language tag from the context.
func LocaleTag(ctx context.Context) (language.Tag, error) {
	lang := locale.Language
	if v, ok := ctx.Value(locale.Locale).(string); ok && v != "" {
		lang = v
	}
	if lang == "" {
		lang = "en"
	}
	return language.Parse(lang)
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
