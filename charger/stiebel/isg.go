package stiebel

import (
	"math"

	"github.com/volkszaehler/mbmd/encoding"
)

type Type int

const (
	_ Type = iota
	Int16
	Uint16
	Bits
)

type Register struct {
	Addr                uint16
	Name, Comment, Unit string
	Typ                 Type
	Divider             float64
}

func Invalid(b []byte) bool {
	return encoding.Int16(b) == math.MinInt16
}

func (reg Register) Float(b []byte) float64 {
	var i int64

	switch reg.Typ {
	case Int16:
		if Invalid(b) {
			return math.NaN()
		}
		i = int64(encoding.Int16(b))
	case Uint16:
		if Invalid(b) {
			return math.NaN()
		}
		i = int64(encoding.Uint16(b))
	default:
		panic("invalid register type")
	}

	f := float64(i)
	if reg.Divider != 0 {
		f = f / reg.Divider
	}

	return f
}
