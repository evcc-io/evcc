package smartgrid

import (
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

var _ export.Writer = (*GridSessions)(nil)

// Write implements the export.Writer interface
func (t *GridSessions) Write(ww export.RowWriter) error {
	return export.WriteStructSlice(ww, t, export.Config{
		I18nPrefix: "config.hems.csv",
	})
}
