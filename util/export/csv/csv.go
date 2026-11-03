package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/util/export"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// rowWriter renders typed cells as locale-independent strings
type rowWriter struct {
	*csv.Writer
	localizer *i18n.Localizer
}

func (w rowWriter) Localizer() *i18n.Localizer { return w.localizer }

func (w rowWriter) Write(record []any) error {
	row := make([]string, len(record))
	for i, v := range record {
		switch v := v.(type) {
		case nil:
			row[i] = ""
		case float64:
			row[i] = strconv.FormatFloat(v, 'f', -1, 64)
		case time.Time:
			row[i] = v.Format("2006-01-02 15:04:05")
		default:
			row[i] = fmt.Sprintf("%v", v)
		}
	}
	return w.Writer.Write(row)
}

// New writes a UTF-8 BOM and returns a csv row writer. The context locale only affects header captions.
func New(ctx context.Context, w io.Writer) (export.RowWriter, error) {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return nil, err
	}

	return rowWriter{csv.NewWriter(w), export.Localizer(ctx)}, nil
}

// WriteStructSlice writes a slice of structs to CSV with localized headers.
func WriteStructSlice(ctx context.Context, w io.Writer, slice any, cfg export.Config) error {
	ww, err := New(ctx, w)
	if err != nil {
		return err
	}
	return export.WriteStructSlice(ww, slice, cfg)
}
