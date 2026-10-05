package export

import (
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type Config struct {
	I18nPrefix string // e.g., "sessions.csv" or "config.hems.csv"
}

// RowWriter is a typed row sink shared by the csv and xlsx exporters
type RowWriter interface {
	Write(record []any) error
	Flush()
	Error() error
	Localizer() *i18n.Localizer
}

// Writer serializes itself to a RowWriter.
type Writer interface {
	Write(RowWriter) error
}
