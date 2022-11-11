package db

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/fatih/structs"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Session is a single charging session
type Session struct {
	ID            uint      `json:"-" csv:"-" gorm:"primarykey"`
	Created       time.Time `json:"created"`
	Finished      time.Time `json:"finished"`
	Loadpoint     string    `json:"loadpoint"`
	Identifier    string    `json:"identifier"`
	Vehicle       string    `json:"vehicle"`
	MeterStart    float64   `json:"meterStart" csv:"Meter Start (kWh)" gorm:"column:meter_start_kwh"`
	MeterStop     float64   `json:"meterStop" csv:"Meter Stop (kWh)" gorm:"column:meter_end_kwh"`
	ChargedEnergy float64   `json:"chargedEnergy" csv:"Charged Energy (kWh)" gorm:"column:charged_kwh"`
}

// Stop stops charging session with end meter reading and due total amount
func (t *Session) Stop(chargedWh, total float64) {
	if chargedEnergy := chargedWh / 1e3; chargedEnergy > t.ChargedEnergy {
		t.ChargedEnergy = chargedEnergy
	}
	t.MeterStop = total
	t.Finished = time.Now()
}

// Sessions is a list of sessions
type Sessions []Session

var _ api.CsvWriter = (*Sessions)(nil)

func (t *Sessions) writeHeader(ctx context.Context, ww *csv.Writer) {
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
	_ = ww.Write(row)
}

func (t *Sessions) writeRow(ww *csv.Writer, r Session) {
	var row []string
	for _, f := range structs.Fields(r) {
		if f.Tag("csv") == "-" {
			continue
		}

		var val string

		switch v := f.Value().(type) {
		case float64:
			val = strconv.FormatFloat(v, 'f', 3, 64)
		case time.Time:
			if !v.IsZero() {
				val = v.Local().Format("2006-01-02 15:04:05")
			}
		default:
			val = fmt.Sprintf("%v", f.Value())
		}

		row = append(row, val)
	}

	_ = ww.Write(row)
}

// WriteCsv implements the api.CsvWriter interface
func (t *Sessions) WriteCsv(ctx context.Context, w io.Writer) {
	ww := csv.NewWriter(w)
	t.writeHeader(ctx, ww)

	for _, r := range *t {
		t.writeRow(ww, r)
	}

	ww.Flush()
}
