package csv

import (
	"context"
	"encoding/csv"
	"io"

	"github.com/evcc-io/evcc/util/export"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// rowWriter adds the localized printer and localizer to a csv.Writer.
type rowWriter struct {
	*csv.Writer
	printer   *message.Printer
	localizer *i18n.Localizer
}

func (w rowWriter) Printer() *message.Printer  { return w.printer }
func (w rowWriter) Localizer() *i18n.Localizer { return w.localizer }

// NewLocalized writes a UTF-8 BOM and returns a locale-configured csv row
// writer (German uses ';').
func NewLocalized(ctx context.Context, w io.Writer) (export.RowWriter, error) {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return nil, err
	}

	tag, err := export.LocaleTag(ctx)
	if err != nil {
		return nil, err
	}

	ww := csv.NewWriter(w)
	if b, _ := tag.Base(); b.String() == language.German.String() {
		ww.Comma = ';'
	}

	return rowWriter{ww, message.NewPrinter(tag), export.Localizer(ctx)}, nil
}

// WriteStructSlice writes a slice of structs to CSV with localized headers.
func WriteStructSlice(ctx context.Context, w io.Writer, slice any, cfg export.Config) error {
	ww, err := NewLocalized(ctx, w)
	if err != nil {
		return err
	}
	return export.WriteStructSlice(ww, slice, cfg)
}
