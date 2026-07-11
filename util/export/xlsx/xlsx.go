package xlsx

import (
	"context"
	"io"

	"github.com/evcc-io/evcc/util/export"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/message"
)

// writer adapts excelize to export.RowWriter, emitting the same localized
// strings as the CSV export.
type writer struct {
	f         *excelize.File
	sheet     string
	dst       io.Writer
	row       int
	err       error
	printer   *message.Printer
	localizer *i18n.Localizer
}

func (x *writer) Write(record []string) error {
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
	if x.row == 1 {
		if err := x.styleHeader(len(record)); err != nil {
			x.err = err
			return err
		}
	}
	x.row++
	return nil
}

// styleHeader makes the header row bold and freezes it below the viewport.
func (x *writer) styleHeader(cols int) error {
	if cols == 0 {
		return nil
	}
	style, err := x.f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	last, err := excelize.CoordinatesToCellName(cols, 1)
	if err != nil {
		return err
	}
	if err := x.f.SetCellStyle(x.sheet, "A1", last, style); err != nil {
		return err
	}
	return x.f.SetPanes(x.sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
}

func (x *writer) Flush() {
	if x.err != nil {
		return
	}
	if err := x.f.Write(x.dst); err != nil {
		x.err = err
	}
}

func (x *writer) Error() error               { return x.err }
func (x *writer) Printer() *message.Printer  { return x.printer }
func (x *writer) Localizer() *i18n.Localizer { return x.localizer }

// NewLocalized returns an xlsx-backed export.RowWriter for the context's locale,
// mirroring csv.NewLocalized.
func NewLocalized(ctx context.Context, w io.Writer) (export.RowWriter, error) {
	tag, err := export.LocaleTag(ctx)
	if err != nil {
		return nil, err
	}
	f := excelize.NewFile()
	return &writer{
		f:         f,
		sheet:     f.GetSheetName(0),
		dst:       w,
		row:       1,
		printer:   message.NewPrinter(tag),
		localizer: export.Localizer(ctx),
	}, nil
}

// WriteStructSlice writes a slice of structs to xlsx with localized headers.
func WriteStructSlice(ctx context.Context, w io.Writer, slice any, cfg export.Config) error {
	ww, err := NewLocalized(ctx, w)
	if err != nil {
		return err
	}
	return export.WriteStructSlice(ww, slice, cfg)
}
