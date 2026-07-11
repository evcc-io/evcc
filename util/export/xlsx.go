package export

import (
	"context"
	"io"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/message"
)

// xlsxWriter adapts excelize to RowWriter, emitting the same localized strings
// as the CSV export. ponytail: numeric-typed cells only if in-sheet math needed.
type xlsxWriter struct {
	f     *excelize.File
	sheet string
	dst   io.Writer
	row   int
	err   error
}

func newXlsxWriter(w io.Writer) *xlsxWriter {
	f := excelize.NewFile()
	return &xlsxWriter{f: f, sheet: f.GetSheetName(0), dst: w, row: 1}
}

func (x *xlsxWriter) Write(record []string) error {
	for i, v := range record {
		cell, err := excelize.CoordinatesToCellName(i+1, x.row)
		if err != nil {
			x.err = err
			return err
		}
		if err := x.f.SetCellValue(x.sheet, cell, v); err != nil {
			x.err = err
			return err
		}
	}
	x.row++
	return nil
}

func (x *xlsxWriter) Flush() {
	if x.err != nil {
		return
	}
	if err := x.f.Write(x.dst); err != nil {
		x.err = err
	}
}

func (x *xlsxWriter) Error() error {
	return x.err
}

// NewLocalizedXlsx returns an xlsx-backed RowWriter and a matching locale
// number printer, mirroring NewLocalizedCsv.
func NewLocalizedXlsx(ctx context.Context, w io.Writer) (RowWriter, *message.Printer, error) {
	tag, err := localeTag(ctx)
	if err != nil {
		return nil, nil, err
	}
	return newXlsxWriter(w), message.NewPrinter(tag), nil
}

// WriteStructSliceXlsx writes a slice of structs to xlsx with localized headers.
func WriteStructSliceXlsx(ctx context.Context, w io.Writer, slice any, cfg Config) error {
	ww, mp, err := NewLocalizedXlsx(ctx, w)
	if err != nil {
		return err
	}
	return writeStructSlice(ctx, ww, mp, slice, cfg)
}
