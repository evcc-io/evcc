package smartgrid

import (
	"context"
	"io"
	"time"

	csvutil "github.com/evcc-io/evcc/util/csv"
)

type GridSession struct {
	ID         uint      `json:"id" csv:"-" gorm:"primarykey"`
	Created    time.Time `json:"created,omitzero"`
	Finished   time.Time `json:"finished,omitzero"`
	Type       Type      `json:"type"`
	GridPower  *float64  `json:"grid,omitempty"`
	LimitPower float64   `json:"limit"`
}

type Type string

const (
	Dim     Type = "consumption"
	Curtail Type = "feedin"
)

type GridSessions []GridSession

// WriteCsv implements the api.CsvWriter interface
func (t *GridSessions) WriteCsv(ctx context.Context, w io.Writer) error {
	return csvutil.WriteStructSlice(ctx, w, t, csvutil.Config{
		I18nPrefix: "config.hems.csv",
	})
}
