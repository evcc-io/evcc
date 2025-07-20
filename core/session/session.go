package session

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/fatih/structs"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// Session is a single charging session
type Session struct {
	ID              uint           `json:"id" csv:"-" gorm:"primarykey"`
	Created         time.Time      `json:"created"`
	Finished        time.Time      `json:"finished"`
	Loadpoint       string         `json:"loadpoint"`
	Identifier      string         `json:"identifier"`
	Vehicle         string         `json:"vehicle"`
	Odometer        *float64       `json:"odometer" format:"int"`
	MeterStart      *float64       `json:"meterStart" csv:"Meter Start (kWh)" gorm:"column:meter_start_kwh"`
	MeterStop       *float64       `json:"meterStop" csv:"Meter Stop (kWh)" gorm:"column:meter_end_kwh"`
	ChargedEnergy   float64        `json:"chargedEnergy" csv:"Charged Energy (kWh)" gorm:"column:charged_kwh"`
	ChargeDuration  *time.Duration `json:"chargeDuration" csv:"Charge Duration" gorm:"column:charge_duration"`
	SolarPercentage *float64       `json:"solarPercentage" csv:"Solar (%)" gorm:"column:solar_percentage"`
	Price           *float64       `json:"price" csv:"Price" gorm:"column:price"`
	PricePerKWh     *float64       `json:"pricePerKWh" csv:"Price/kWh" gorm:"column:price_per_kwh"`
	Co2PerKWh       *float64       `json:"co2PerKWh" csv:"CO2/kWh (gCO2eq)" gorm:"column:co2_per_kwh"`
}

// Sessions is a list of sessions
type Sessions []Session

var _ api.CsvWriter = (*Sessions)(nil)

func (t *Sessions) writeHeader(ctx context.Context, ww *csv.Writer) error {
	localizer := locale.Localizer
	if val := ctx.Value(locale.Locale).(string); val != "" {
		localizer = i18n.NewLocalizer(locale.Bundle, val, locale.Language)
	}

	var row []string
	for _, f := range structs.Fields(Session{}) {
		csv := f.Tag("csv")
		if csv == "-" {
			continue
		}

		caption, err := localizer.Localize(&locale.Config{
			MessageID: "sessions.csv." + strings.ToLower(f.Name()),
		})
		if err != nil {
			if csv != "" {
				caption = csv
			} else {
				caption = f.Name()
			}
		}

		row = append(row, caption)
	}

	return ww.Write(row)
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

func (t *Sessions) writeRow(ww *csv.Writer, mp *message.Printer, r Session) error {
	var row []string
	for _, f := range structs.Fields(r) {
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

// WriteCsv implements the api.CsvWriter interface
func (t *Sessions) WriteCsv(ctx context.Context, w io.Writer) error {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	// get context language
	lang := locale.Language
	if language, ok := ctx.Value(locale.Locale).(string); ok && language != "" {
		lang = language
	}

	tag, err := language.Parse(lang)
	if err != nil {
		return err
	}

	ww := csv.NewWriter(w)

	// set separator according to locale
	if b, _ := tag.Base(); b.String() == language.German.String() {
		ww.Comma = ';'
	}

	if err := t.writeHeader(ctx, ww); err != nil {
		return err
	}

	mp := message.NewPrinter(tag)
	for _, r := range *t {
		if err := t.writeRow(ww, mp, r); err != nil {
			return err
		}
	}

	ww.Flush()

	return ww.Error()
}
