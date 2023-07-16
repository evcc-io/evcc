package db

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
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

/*
	Per Row arround 200 byte (data + indices)
	Snapshot per minute = 1440 snapshots per day
	-> 281,25 KB per day
	-> 100,25 MB per year
	-> 1,95 GB per 20 years
*/
// PowerState is a single power measurement snapshot
type PowerState struct {
	ID          uint    `json:"-" csv:"-" gorm:"primarykey"`
	Year        uint16  `json:"year" gorm:"index"`
	Month       uint16  `json:"month" gorm:"index"`
	Day         uint16  `json:"day" gorm:"index"`
	Hour        uint16  `json:"Hour" gorm:"index"`
	Minute      uint16  `json:"minute" gorm:"index"`
	FromPvs     float64 `json:"FromPvs"`
	FromStorage float64 `json:"FromStorage"`
	FromGrid    float64 `json:"FromGrid"`
	ToGrid      float64 `json:"ToGrid"`
	ToStorage   float64 `json:"ToStorage"`
	ToHouse     float64 `json:"ToHouse"`
	ToCars      float64 `json:"ToCars"`
	ToHeating   float64 `json:"ToHeating"`
	BatterySoC  float64 `json:"BatterySoC"`
}

// Power is a list of PowerState
type Power []PowerState

var _ api.CsvWriter = (*Power)(nil)

func (t *Power) writeHeader(ctx context.Context, ww *csv.Writer) error {
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

func (t *Power) writeRow(ww *csv.Writer, mp *message.Printer, r PowerState) error {
	var row []string
	for _, f := range structs.Fields(r) {
		if f.Tag("csv") == "-" {
			continue
		}

		var val string
		format := f.Tag("format")

		switch v := f.Value().(type) {
		case float64:
			switch format {
			case "int":
				val = mp.Sprint(number.Decimal(v, number.NoSeparator(), number.MaxFractionDigits(0)))
			default:
				val = mp.Sprint(number.Decimal(v, number.NoSeparator(), number.MaxFractionDigits(3)))
			}
		case time.Time:
			if !v.IsZero() {
				val = v.Local().Format("2006-01-02 15:04:05")
			}
		default:
			val = fmt.Sprintf("%v", f.Value())
		}

		row = append(row, val)
	}

	return ww.Write(row)
}

// WriteCsv implements the api.CsvWriter interface
func (t *Power) WriteCsv(ctx context.Context, w io.Writer) error {
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
