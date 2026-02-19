package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
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

type Config struct {
	I18nPrefix string // e.g., "sessions.csv" or "config.hems.csv"
}

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

func writeHeader(ctx context.Context, ww *csv.Writer, structType any, i18nPrefix string) error {
	localizer := locale.Localizer
	if val, ok := ctx.Value(locale.Locale).(string); ok && val != "" {
		localizer = i18n.NewLocalizer(locale.Bundle, val, locale.Language)
	}

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

func writeRow(ww *csv.Writer, mp *message.Printer, structVal any) error {
	var row []string
	for _, f := range structs.Fields(structVal) {
		if f.Tag("csv") == "-" {
			continue
		}

		digits := 3
		if format := f.Tag("format"); format == "int" {
			digits = 0
		}

		val := formatValue(mp, f.Value(), digits)
		row = append(row, val)
	}

	return ww.Write(row)
}

// WriteStructSlice writes a slice of structs to CSV format with localized headers
func WriteStructSlice(ctx context.Context, w io.Writer, slice any, cfg Config) error {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	lang := locale.Language
	if language, ok := ctx.Value(locale.Locale).(string); ok && language != "" {
		lang = language
	}
	if lang == "" {
		lang = "en"
	}

	tag, err := language.Parse(lang)
	if err != nil {
		return err
	}

	ww := csv.NewWriter(w)

	if b, _ := tag.Base(); b.String() == language.German.String() {
		ww.Comma = ';'
	}

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

	if err := writeHeader(ctx, ww, structType, cfg.I18nPrefix); err != nil {
		return err
	}

	mp := message.NewPrinter(tag)
	for i := 0; i < sliceVal.Len(); i++ {
		row := sliceVal.Index(i).Interface()
		if err := writeRow(ww, mp, row); err != nil {
			return err
		}
	}

	ww.Flush()
	return ww.Error()
}
