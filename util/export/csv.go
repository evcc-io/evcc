package export

import (
	"context"
	"encoding/csv"
	"io"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// NewLocalizedCsv writes a UTF-8 BOM and returns a locale-configured
// csv.Writer (German uses ';') and matching number printer.
func NewLocalizedCsv(ctx context.Context, w io.Writer) (*csv.Writer, *message.Printer, error) {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return nil, nil, err
	}

	tag, err := localeTag(ctx)
	if err != nil {
		return nil, nil, err
	}

	ww := csv.NewWriter(w)
	if b, _ := tag.Base(); b.String() == language.German.String() {
		ww.Comma = ';'
	}

	return ww, message.NewPrinter(tag), nil
}

// WriteStructSlice writes a slice of structs to CSV format with localized headers
func WriteStructSlice(ctx context.Context, w io.Writer, slice any, cfg Config) error {
	ww, mp, err := NewLocalizedCsv(ctx, w)
	if err != nil {
		return err
	}
	return writeStructSlice(ctx, ww, mp, slice, cfg)
}
