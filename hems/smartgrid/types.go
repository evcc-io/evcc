package smartgrid

import (
	"time"
)

type GridSession struct {
	ID         uint      `json:"id" csv:"-" gorm:"primarykey"`
	Created    time.Time `json:"created,omitzero"`
	Finished   time.Time `json:"finished,omitzero"`
	Type       Type      `json:"type"`
	GridPower  *float64  `json:"grid,omitempty"`
	LimitPower float64   `json:"limit"`
}

type Type rune

const (
	Dim     Type = 'D'
	Curtail Type = 'C'
)
