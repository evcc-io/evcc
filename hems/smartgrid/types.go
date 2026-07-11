package smartgrid

import (
	"context"
	"io"
	"time"

	"github.com/evcc-io/evcc/util/export"
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
	Curtail Type = "production"
)

type GridSessions []GridSession

// WriteCsv implements the api.CsvWriter interface
func (t *GridSessions) WriteCsv(ctx context.Context, w io.Writer) error {
	return export.WriteStructSlice(ctx, w, t, export.Config{
		I18nPrefix: "config.hems.csv",
	})
}

// WriteXlsx implements the api.XlsxWriter interface
func (t *GridSessions) WriteXlsx(ctx context.Context, w io.Writer) error {
	return export.WriteStructSliceXlsx(ctx, w, t, export.Config{
		I18nPrefix: "config.hems.csv",
	})
}
